package controller

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/modeliqcheck"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

const iqCheckPrompt = "Write a landing page as a single HTML file."

func TestApplyChannelTestOptionsReplacesChatPrompt(t *testing.T) {
	request := buildTestRequest("gpt-5", "", nil, true, common.GetPointer(uint(8192)))

	require.NoError(t, applyChannelTestOptions(request, channelTestOptions{prompt: iqCheckPrompt}))

	chat, ok := request.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chat.Messages, 1)
	assert.Equal(t, "user", chat.Messages[0].Role)
	assert.Equal(t, iqCheckPrompt, chat.Messages[0].Content)
	require.NotNil(t, chat.MaxTokens, "a configured output budget is kept")
	assert.Equal(t, uint(8192), *chat.MaxTokens)
	assert.True(t, *chat.Stream)
	require.NotNil(t, chat.StreamOptions, "streamed checks still ask for usage")
}

func TestApplyChannelTestOptionsReplacesResponsesInput(t *testing.T) {
	request := buildTestRequest("gpt-5-codex", "", nil, true, nil)

	require.NoError(t, applyChannelTestOptions(request, channelTestOptions{prompt: "say \"hi\"\nthen stop"}))

	responses, ok := request.(*dto.OpenAIResponsesRequest)
	require.True(t, ok)
	var input []map[string]string
	require.NoError(t, common.Unmarshal(responses.Input, &input))
	assert.Equal(t, []map[string]string{{"role": "user", "content": "say \"hi\"\nthen stop"}}, input)
}

func TestApplyChannelTestOptionsReplacesNativePrompts(t *testing.T) {
	gemini := buildTestRequest("gemini-2.5-pro", string(constant.EndpointTypeGemini), nil, true, nil)
	require.NoError(t, applyChannelTestOptions(gemini, channelTestOptions{prompt: iqCheckPrompt}))
	geminiReq, ok := gemini.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.Len(t, geminiReq.Contents, 1)
	assert.Equal(t, "user", geminiReq.Contents[0].Role)
	require.Len(t, geminiReq.Contents[0].Parts, 1)
	assert.Equal(t, iqCheckPrompt, geminiReq.Contents[0].Parts[0].Text)

	claude := buildTestRequest("claude-sonnet-4", string(constant.EndpointTypeAnthropic), nil, true, nil)
	require.NoError(t, applyChannelTestOptions(claude, channelTestOptions{prompt: iqCheckPrompt, omitMaxTokens: true}))
	claudeReq, ok := claude.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.Len(t, claudeReq.Messages, 1)
	assert.Equal(t, iqCheckPrompt, claudeReq.Messages[0].Content)
	// Claude upstreams get the Claude default max tokens after conversion, while
	// the same request converted for other upstreams carries no limit.
	assert.Nil(t, claudeReq.MaxTokens)
}

func TestApplyChannelTestOptionsOmitsMaxTokens(t *testing.T) {
	cases := []struct {
		name  string
		model string
	}{
		{name: "max_tokens model", model: "gpt-4o"},
		{name: "reasoning model uses max_completion_tokens", model: "o3-mini"},
		{name: "gemini via openai format", model: "gemini-2.5-pro"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := buildTestRequest(testCase.model, "", nil, true, nil)
			require.NoError(t, applyChannelTestOptions(request, channelTestOptions{prompt: iqCheckPrompt, omitMaxTokens: true}))
			chat, ok := request.(*dto.GeneralOpenAIRequest)
			require.True(t, ok)
			assert.Nil(t, chat.MaxTokens)
			assert.Nil(t, chat.MaxCompletionTokens)
		})
	}

	native := buildTestRequest("gemini-2.5-pro", string(constant.EndpointTypeGemini), nil, true, nil)
	require.NoError(t, applyChannelTestOptions(native, channelTestOptions{omitMaxTokens: true}))
	geminiReq, ok := native.(*dto.GeminiChatRequest)
	require.True(t, ok)
	assert.Nil(t, geminiReq.GenerationConfig.MaxOutputTokens)
}

// A configured max tokens must reach the upstream for every text format, also
// where the synthetic probe leaves the field out: an HTML answer cut at the
// upstream default reads as a dumber model.
func TestApplyChannelTestOptionsSendsConfiguredMaxTokens(t *testing.T) {
	maxTokens := common.GetPointer(uint(32000))

	thinking := buildTestRequest("claude-sonnet-4-5-thinking", "", nil, true, maxTokens)
	require.NoError(t, applyChannelTestOptions(thinking, channelTestOptions{prompt: iqCheckPrompt, maxTokens: maxTokens}))
	chat, ok := thinking.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotNil(t, chat.MaxTokens)
	assert.Equal(t, uint(32000), *chat.MaxTokens)

	responses := buildTestRequest("gpt-5-codex", "", nil, true, maxTokens)
	require.NoError(t, applyChannelTestOptions(responses, channelTestOptions{prompt: iqCheckPrompt, maxTokens: maxTokens}))
	responsesReq, ok := responses.(*dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.NotNil(t, responsesReq.MaxOutputTokens)
	assert.Equal(t, uint(32000), *responsesReq.MaxOutputTokens)

	// The channel monitor probe sends no prompt: a 16-token cap would be
	// rejected by claude thinking models, so the probe keeps leaving it out.
	probeTokens := common.GetPointer(uint(16))
	probe := buildTestRequest("claude-sonnet-4-5-thinking", "", nil, true, probeTokens)
	require.NoError(t, applyChannelTestOptions(probe, channelTestOptions{maxTokens: probeTokens}))
	probeReq, ok := probe.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	assert.Nil(t, probeReq.MaxTokens)
}

func TestApplyChannelTestOptionsRejectsPromptForNonChatRequests(t *testing.T) {
	for _, testCase := range []struct {
		model        string
		endpointType string
	}{
		{model: "text-embedding-3-large"},
		{model: "jina-rerank-v2"},
		{model: "gpt-image-1", endpointType: string(constant.EndpointTypeImageGeneration)},
	} {
		request := buildTestRequest(testCase.model, testCase.endpointType, nil, true, nil)
		err := applyChannelTestOptions(request, channelTestOptions{prompt: iqCheckPrompt})
		require.Error(t, err, "model %s", testCase.model)
		assert.Contains(t, err.Error(), "does not accept a text prompt")
	}
}

func TestApplyChannelTestOptionsWithoutOptionsKeepsSyntheticProbe(t *testing.T) {
	request := buildTestRequest("gpt-4o", "", nil, true, nil)
	require.NoError(t, applyChannelTestOptions(request, channelTestOptions{group: "vip", skipConsumeLog: true}))

	chat, ok := request.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Len(t, chat.Messages, 1)
	assert.Equal(t, "hi", chat.Messages[0].Content)
	require.NotNil(t, chat.MaxTokens)
	assert.Equal(t, uint(16), *chat.MaxTokens)

	embedding := buildTestRequest("text-embedding-3-large", "", nil, false, nil)
	assert.NoError(t, applyChannelTestOptions(embedding, channelTestOptions{}), "plain channel tests of non-chat models must keep working")
}

func TestReadTestResponseBodyCapturesWholeStream(t *testing.T) {
	stream := strings.Repeat("data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\n", 1000)
	require.Greater(t, len(stream), 8<<10)

	logged, err := readTestResponseBody(io.NopCloser(strings.NewReader(stream)), true, false)
	require.NoError(t, err)
	assert.Len(t, logged, 8<<10, "plain channel tests only keep a log-sized prefix")

	captured, err := readTestResponseBody(io.NopCloser(strings.NewReader(stream)), true, true)
	require.NoError(t, err)
	assert.Equal(t, stream, string(captured))
}

func TestGetModelIQCheckResultsValidatesIds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		_ = config.GlobalConfig.LoadFromDB(saved)
	})
	// With the feature disabled, valid requests return an empty list without
	// reading the database, which isolates the id parsing contract.
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		model_iq_check_setting.EnabledOptionKey: "false",
	}))

	atCap := make([]string, 0, maxModelIQCheckResultIds*2)
	overCap := make([]string, 0, maxModelIQCheckResultIds+1)
	for id := 1; id <= maxModelIQCheckResultIds; id++ {
		atCap = append(atCap, strconv.Itoa(id), strconv.Itoa(id))
		overCap = append(overCap, strconv.Itoa(id))
	}
	overCap = append(overCap, strconv.Itoa(maxModelIQCheckResultIds+1))

	cases := []struct {
		name        string
		query       string
		wantSuccess bool
	}{
		{name: "missing ids", query: "", wantSuccess: true},
		{name: "blank segments", query: "ids=,%20,", wantSuccess: true},
		{name: "valid ids", query: "ids=1,2,3", wantSuccess: true},
		{name: "duplicates collapse to the cap", query: "ids=" + strings.Join(atCap, ","), wantSuccess: true},
		{name: "malformed id", query: "ids=1,abc", wantSuccess: false},
		{name: "zero id", query: "ids=0", wantSuccess: false},
		{name: "negative id", query: "ids=-5", wantSuccess: false},
		{name: "too many ids", query: "ids=" + strings.Join(overCap, ","), wantSuccess: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/iq-check/results?"+testCase.query, nil)

			GetModelIQCheckResults(c)

			var response struct {
				Success bool             `json:"success"`
				Message string           `json:"message"`
				Data    []map[string]any `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, testCase.wantSuccess, response.Success, response.Message)
			if testCase.wantSuccess {
				assert.NotNil(t, response.Data, "an empty result must serialize as []")
				assert.Empty(t, response.Data)
			} else {
				assert.NotEmpty(t, response.Message)
			}
		})
	}
}

func setupModelIQCheckConfigTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Log{}, &model.ModelIQCheckResult{}, &model.ModelIQCheckState{}, &model.SystemTask{}))
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "gpt-5", ChannelId: 1, Enabled: true}).Error)

	previousOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		common.OptionMap = previousOptionMap
		_ = config.GlobalConfig.LoadFromDB(saved)
		modeliqcheck.InvalidateStatusCache()
	})
	return db
}

func putModelIQCheckConfig(t *testing.T, body any) (bool, string) {
	t.Helper()
	payload, err := common.Marshal(body)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/iq-check/config", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateModelIQCheckConfig(c)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response.Success, response.Message
}

// Saving the config must queue a fresh run only for checks whose answer would
// change, drop the history of removed checks, and wake the scheduler.
func TestUpdateModelIQCheckConfigQueuesChangedChecksAndDropsRemovedData(t *testing.T) {
	db := setupModelIQCheckConfigTest(t)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		model_iq_check_setting.EnabledOptionKey: "true",
		model_iq_check_setting.ChecksOptionKey: `[
			{"id":"edited","name":"Edited","group":"default","model":"gpt-5","prompt":"old prompt","interval_minutes":60,"icon":"OpenAI","enabled":true,"sort":0},
			{"id":"same","name":"Same","group":"default","model":"gpt-5","prompt":"1+1=?","interval_minutes":60,"icon":"OpenAI","enabled":true,"sort":1},
			{"id":"removed","name":"Removed","group":"default","model":"gpt-5","prompt":"p","interval_minutes":60,"icon":"OpenAI","enabled":true,"sort":2}
		]`,
	}))
	for _, checkId := range []string{"edited", "same", "removed"} {
		require.NoError(t, model.MarkModelIQCheckStarted(checkId, 100))
		require.NoError(t, model.SaveModelIQCheckResult(&model.ModelIQCheckResult{CheckId: checkId, CheckedAt: 160, Status: model.ModelIQCheckStatusSuccess}, 10))
	}

	success, message := putModelIQCheckConfig(t, map[string]any{
		"checks": []modeliqcheck.Check{
			{Id: "edited", Name: "Edited", Group: "default", Model: "gpt-5", Prompt: "new prompt", IntervalMinutes: 60, Icon: "OpenAI", Enabled: true},
			{Id: "same", Name: "Same renamed", Group: "default", Model: "gpt-5", Prompt: "1+1=?", IntervalMinutes: 30, Icon: "OpenAI", Enabled: true},
			{Name: "Added", Group: "default", Model: "gpt-5", Prompt: "Write a page", Enabled: true},
		},
	})
	require.True(t, success, message)

	checks := modeliqcheck.Checks()
	require.Len(t, checks, 3)
	added := checks[2]
	assert.NotEmpty(t, added.Id)
	assert.Equal(t, modeliqcheck.DefaultIntervalMinutes, added.IntervalMinutes)

	states, err := model.GetModelIQCheckStates([]string{"edited", "same", "removed", added.Id})
	require.NoError(t, err)
	stateById := make(map[string]model.ModelIQCheckState, len(states))
	for _, state := range states {
		stateById[state.CheckId] = state
	}
	assert.Greater(t, stateById["edited"].RunRequestedAt, int64(100), "a changed prompt queues a fresh answer")
	assert.Zero(t, stateById["same"].RunRequestedAt, "renaming or retiming does not change the answer")
	assert.Greater(t, stateById[added.Id].RunRequestedAt, int64(0), "a new check is queued")
	_, removedStateLeft := stateById["removed"]
	assert.False(t, removedStateLeft)

	history, err := model.GetModelIQCheckHistory([]string{"edited", "same", "removed"})
	require.NoError(t, err)
	require.Len(t, history, 2)
	for _, row := range history {
		assert.NotEqual(t, "removed", row.CheckId)
	}

	var tasks int64
	require.NoError(t, db.Model(&model.SystemTask{}).Where("type = ?", model.SystemTaskTypeModelIQCheck).Count(&tasks).Error)
	assert.Equal(t, int64(1), tasks, "queued runs wake the scheduler")
}

func TestUpdateModelIQCheckConfigRejectsInvalidInputWithoutSaving(t *testing.T) {
	setupModelIQCheckConfigTest(t)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		model_iq_check_setting.EnabledOptionKey:      "false",
		model_iq_check_setting.HistoryLimitOptionKey: "10",
		model_iq_check_setting.ChecksOptionKey:       `[]`,
	}))

	cases := []struct {
		name string
		body map[string]any
	}{
		{name: "history limit below range", body: map[string]any{"enabled": true, "history_limit": 0}},
		{name: "history limit above range", body: map[string]any{"enabled": true, "history_limit": model_iq_check_setting.MaxHistoryLimit + 1}},
		{name: "timeout below range", body: map[string]any{"enabled": true, "request_timeout_seconds": model_iq_check_setting.MinRequestTimeoutSeconds - 1}},
		{name: "model not in group", body: map[string]any{"enabled": true, "checks": []modeliqcheck.Check{{Name: "Pro", Group: "default", Model: "unknown-model", Prompt: "p"}}}},
		{name: "unknown group", body: map[string]any{"enabled": true, "checks": []modeliqcheck.Check{{Name: "Pro", Group: "ghost", Model: "gpt-5", Prompt: "p"}}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			success, message := putModelIQCheckConfig(t, testCase.body)
			assert.False(t, success)
			assert.NotEmpty(t, message)
			assert.False(t, model_iq_check_setting.IsEnabled(), "a rejected request must not apply any part of the update")
			assert.Equal(t, 10, model_iq_check_setting.GetSetting().HistoryLimit)
			assert.Empty(t, modeliqcheck.Checks())
		})
	}
}

// A check can wait in the queue for minutes, and the admin's save may land on
// another node that this one has not synced with yet. The runner must start
// each check from the config stored in the database, so a queued re-run is not
// spent on the old prompt and removed checks cost no upstream calls.
func TestRunModelIQCheckStartsFromStoredConfig(t *testing.T) {
	db := setupModelIQCheckConfigTest(t)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		model_iq_check_setting.EnabledOptionKey: "true",
		model_iq_check_setting.ChecksOptionKey: `[
			{"id":"edited","name":"Edited","group":"default","model":"gpt-5","prompt":"old prompt","interval_minutes":60,"enabled":true,"sort":0},
			{"id":"removed","name":"Removed","group":"default","model":"gpt-5","prompt":"p","interval_minutes":60,"enabled":true,"sort":1}
		]`,
	}))
	queued := modeliqcheck.Checks()
	require.Len(t, queued, 2)
	require.NoError(t, db.Create(&model.Option{
		Key:   model_iq_check_setting.ChecksOptionKey,
		Value: `[{"id":"edited","name":"Edited","group":"default","model":"gpt-5","prompt":"new prompt","interval_minutes":60,"enabled":true,"sort":0}]`,
	}).Error)
	setting := model_iq_check_setting.GetSetting()

	assert.Zero(t, runModelIQCheck(context.Background(), queued[1], 1, setting), "a removed check is skipped")
	// No channel serves the model in this fixture, so the run fails right after
	// it starts, which is enough to see which config it used.
	assert.Equal(t, model.ModelIQCheckStatusFailed, runModelIQCheck(context.Background(), queued[0], 1, setting))

	var results []model.ModelIQCheckResult
	require.NoError(t, db.Find(&results).Error)
	require.Len(t, results, 1)
	assert.Equal(t, "edited", results[0].CheckId)
	assert.Equal(t, "new prompt", results[0].Prompt)
	assert.Equal(t, modeliqcheck.ErrorTypeNoChannel, results[0].ErrorType)

	states, err := model.GetModelIQCheckStates([]string{"edited", "removed"})
	require.NoError(t, err)
	require.Len(t, states, 1, "a skipped check must not be marked as started")
	assert.Equal(t, "edited", states[0].CheckId)
}

// Writing the checks option directly must store the normalized config: a
// check without an id could never record its run and would be due forever.
func TestUpdateOptionStoresNormalizedModelIQChecks(t *testing.T) {
	setupModelIQCheckConfigTest(t)
	payload, err := common.Marshal(map[string]any{
		"key":   model_iq_check_setting.ChecksOptionKey,
		"value": `[{"name":" Pro ","group":"default","model":"gpt-5","prompt":"Write a landing page","enabled":true}]`,
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateOption(c)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	checks := modeliqcheck.Checks()
	require.Len(t, checks, 1)
	assert.NotEmpty(t, checks[0].Id)
	assert.Equal(t, "Pro", checks[0].Name)
	assert.Equal(t, modeliqcheck.DefaultIntervalMinutes, checks[0].IntervalMinutes)
}

// setupModelIQCheckChannel stores a root user and a channel serving gpt-5 in
// the default group, so executeModelIQCheck runs the real channel test path.
func setupModelIQCheckChannel(t *testing.T, channelType int, baseURL string, modelMapping string) *model.Channel {
	t.Helper()
	db := setupModelIQCheckConfigTest(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1,
		Username: "root",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Setting:  `{"accept_unset_model_ratio_model":true}`,
	}).Error)
	channel := &model.Channel{
		Id:      1,
		Type:    channelType,
		Key:     "sk-test",
		Name:    "iq",
		Status:  common.ChannelStatusEnabled,
		Models:  "gpt-5",
		Group:   "default",
		BaseURL: &baseURL,
	}
	if modelMapping != "" {
		channel.ModelMapping = &modelMapping
	}
	require.NoError(t, db.Create(channel).Error)
	return channel
}

// iqCheckChunk is one OpenAI chat completion stream event; a non-empty
// finishReason marks the model's last chunk.
func iqCheckChunk(content string, finishReason string) string {
	finish := "null"
	if finishReason != "" {
		finish = strconv.Quote(finishReason)
	}
	return `data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-5","choices":[{"index":0,"delta":{"content":` + strconv.Quote(content) + `},"finish_reason":` + finish + `}]}` + "\n\n"
}

// writeChunkedThenDrop answers with a chunked 200 that carries body and then
// drops the connection without the terminating chunk, like a reset mid-stream.
func writeChunkedThenDrop(t *testing.T, w http.ResponseWriter, contentType string, body string) {
	conn, buffered, err := w.(http.Hijacker).Hijack()
	if !assert.NoError(t, err) {
		return
	}
	defer conn.Close()
	_, _ = io.WriteString(buffered, "HTTP/1.1 200 OK\r\nContent-Type: "+contentType+"\r\nTransfer-Encoding: chunked\r\n\r\n")
	_, _ = io.WriteString(buffered, strconv.FormatInt(int64(len(body)), 16)+"\r\n"+body+"\r\n")
	_ = buffered.Flush()
}

func runIQCheckAgainst(t *testing.T, channelType int, upstream http.HandlerFunc) model.ModelIQCheckResult {
	t.Helper()
	server := httptest.NewServer(upstream)
	t.Cleanup(server.Close)
	channel := setupModelIQCheckChannel(t, channelType, server.URL, "")
	result := model.ModelIQCheckResult{Status: model.ModelIQCheckStatusFailed}
	executeModelIQCheck(context.Background(), modeliqcheck.Check{Id: "c1", Group: "default", Model: "gpt-5", Prompt: "2+2?"}, channel, 1, 600, &result)
	return result
}

func setStreamingTimeout(t *testing.T, seconds int) {
	t.Helper()
	previous := constant.StreamingTimeout
	constant.StreamingTimeout = seconds
	t.Cleanup(func() { constant.StreamingTimeout = previous })
}

// A reasoning model can stay silent far longer than STREAMING_TIMEOUT before it
// answers; the check must keep waiting until its own timeout.
func TestExecuteModelIQCheckOutlastsStreamingIdleTimeout(t *testing.T) {
	setStreamingTimeout(t, 1)

	var streamed atomic.Bool
	result := runIQCheckAgainst(t, constant.ChannelTypeOpenAI, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		streamed.Store(strings.Contains(string(body), `"stream":true`))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, iqCheckChunk("", ""))
		w.(http.Flusher).Flush()
		time.Sleep(2 * time.Second)
		_, _ = io.WriteString(w, iqCheckChunk("4", "stop"))
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	})

	assert.True(t, streamed.Load(), "the upstream request must ask for a stream")
	assert.Equal(t, model.ModelIQCheckStatusSuccess, result.Status, result.ErrorMessage)
	assert.Equal(t, "4", result.Content)
	assert.Equal(t, "stop", result.FinishReason)
}

// Channels that write through gin's c.Stream (Cohere, Zhipu, Xunfei) must run
// on the channel test recorder instead of panicking.
func TestExecuteModelIQCheckStreamsThroughGinStreamChannels(t *testing.T) {
	setStreamingTimeout(t, 300)

	result := runIQCheckAgainst(t, constant.ChannelTypeCohere, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"is_finished":false,"event_type":"text-generation","text":"4"}`+"\n")
		_, _ = io.WriteString(w, `{"is_finished":true,"event_type":"stream-end","finish_reason":"COMPLETE","response":{"meta":{"billed_units":{"input_tokens":3,"output_tokens":1}}}}`+"\n")
	})

	assert.Equal(t, model.ModelIQCheckStatusSuccess, result.Status, result.ErrorMessage)
	assert.Equal(t, "4", result.Content)
}

// A stream that ends before the model finished is a failed check that keeps
// the partial answer, not a success with half an answer, whichever way the
// channel's handler reports the end.
func TestExecuteModelIQCheckFailsWhenUpstreamStreamEndsEarly(t *testing.T) {
	setStreamingTimeout(t, 300)

	tests := []struct {
		name        string
		channelType int
		upstream    http.HandlerFunc
		wantMessage string
	}{
		{
			name:        "connection dropped",
			channelType: constant.ChannelTypeOpenAI,
			upstream: func(w http.ResponseWriter, _ *http.Request) {
				writeChunkedThenDrop(t, w, "text/event-stream", iqCheckChunk("partial answer", ""))
			},
			wantMessage: "stream ended before completion",
		},
		{
			name:        "closed without a finish reason",
			channelType: constant.ChannelTypeOpenAI,
			upstream: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, iqCheckChunk("partial answer", ""))
			},
			wantMessage: "without a finish reason",
		},
		{
			name:        "handler that synthesizes its own end",
			channelType: constant.ChannelTypeCohere,
			upstream: func(w http.ResponseWriter, _ *http.Request) {
				writeChunkedThenDrop(t, w, "application/json", `{"is_finished":false,"event_type":"text-generation","text":"partial answer"}`+"\n")
			},
			wantMessage: "without a finish reason",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := runIQCheckAgainst(t, test.channelType, test.upstream)

			assert.Equal(t, model.ModelIQCheckStatusFailed, result.Status)
			assert.Equal(t, modeliqcheck.ErrorTypeUpstream, result.ErrorType)
			assert.Contains(t, result.ErrorMessage, test.wantMessage)
			assert.Equal(t, "partial answer", result.Content)
		})
	}
}

// Channels that can only answer in one piece cannot be checked, since the check
// must stream from the upstream model.
func TestExecuteModelIQCheckRejectsChannelsThatCannotStream(t *testing.T) {
	tests := []struct {
		name         string
		channelType  int
		modelMapping string
	}{
		{name: "PaLM", channelType: constant.ChannelTypePaLM},
		{name: "AWS Nova", channelType: constant.ChannelTypeAws, modelMapping: `{"gpt-5":"amazon.nova-pro-v1:0"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("the upstream must not be called")
			}))
			t.Cleanup(upstream.Close)
			channel := setupModelIQCheckChannel(t, test.channelType, upstream.URL, test.modelMapping)
			if test.channelType == constant.ChannelTypeAws {
				channel.Key = "ak|sk|us-east-1"
			}

			result := model.ModelIQCheckResult{Status: model.ModelIQCheckStatusFailed}
			executeModelIQCheck(context.Background(), modeliqcheck.Check{Id: "c1", Group: "default", Model: "gpt-5", Prompt: "2+2?"}, channel, 1, 600, &result)

			assert.Equal(t, model.ModelIQCheckStatusFailed, result.Status)
			assert.Equal(t, modeliqcheck.ErrorTypeUnsupportedModel, result.ErrorType, result.ErrorMessage)
		})
	}
}

// claudeStream is a minimal Anthropic Messages stream with summarized thinking.
const claudeStream = "event: message_start\n" +
	`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[],"usage":{"input_tokens":5,"output_tokens":1}}}` + "\n\n" +
	"event: content_block_start\n" +
	`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}` + "\n\n" +
	"event: content_block_delta\n" +
	`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Add."}}` + "\n\n" +
	"event: content_block_stop\n" +
	`data: {"type":"content_block_stop","index":0}` + "\n\n" +
	"event: content_block_start\n" +
	`data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}` + "\n\n" +
	"event: content_block_delta\n" +
	`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"4"}}` + "\n\n" +
	"event: content_block_stop\n" +
	`data: {"type":"content_block_stop","index":1}` + "\n\n" +
	"event: message_delta\n" +
	`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}` + "\n\n" +
	"event: message_stop\n" +
	`data: {"type":"message_stop"}` + "\n\n"

// The reasoning effort reaches the upstream in the parameters the upstream
// model accepts, whichever request format the check uses, and every response
// format is read back into the answer and its thinking.
func TestExecuteModelIQCheckSendsReasoningForUpstreamModel(t *testing.T) {
	setStreamingTimeout(t, 300)

	responsesStream := "event: response.reasoning_summary_text.delta\n" +
		`data: {"type":"response.reasoning_summary_text.delta","delta":"Add."}` + "\n\n" +
		"event: response.output_text.delta\n" +
		`data: {"type":"response.output_text.delta","delta":"4"}` + "\n\n" +
		"event: response.completed\n" +
		`data: {"type":"response.completed","response":{"id":"r1","status":"completed","output":[],"usage":{"input_tokens":5,"output_tokens":3,"total_tokens":8}}}` + "\n\n"
	geminiStream := `data: {"candidates":[{"content":{"role":"model","parts":[{"text":"Add.","thought":true}]},"index":0}],"usageMetadata":{"promptTokenCount":5}}` + "\n\n" +
		`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"4"}]},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":1,"totalTokenCount":6}}` + "\n\n"
	chatStream := `data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-5","choices":[{"index":0,"delta":{"reasoning_content":"Add."},"finish_reason":null}]}` + "\n\n" +
		iqCheckChunk("4", "stop") + "data: [DONE]\n\n"

	tests := []struct {
		name          string
		channelType   int
		modelMapping  string
		check         modeliqcheck.Check
		wantPath      string
		wantBody      map[string]any
		absentFields  []string
		response      string
		wantReasoning string
	}{
		{
			name:         "claude format on an anthropic channel uses adaptive thinking",
			channelType:  constant.ChannelTypeAnthropic,
			modelMapping: `{"gpt-5":"claude-opus-4-8"}`,
			check:        modeliqcheck.Check{EndpointType: "anthropic", ReasoningEffort: "high", ReasoningSummary: true},
			wantPath:     "/v1/messages",
			wantBody: map[string]any{
				"model": "claude-opus-4-8", "stream": true, "max_tokens": 8192.0,
				"thinking.type": "adaptive", "thinking.display": "summarized", "output_config.effort": "high",
			},
			absentFields:  []string{"thinking.budget_tokens", "temperature"},
			response:      claudeStream,
			wantReasoning: "Add.",
		},
		{
			name:         "chat format on an anthropic channel replaces the fixed thinking budget",
			channelType:  constant.ChannelTypeAnthropic,
			modelMapping: `{"gpt-5":"claude-opus-4-8"}`,
			check:        modeliqcheck.Check{ReasoningEffort: "xhigh", MaxTokens: 64000},
			wantPath:     "/v1/messages",
			wantBody:     map[string]any{"max_tokens": 64000.0, "thinking.type": "adaptive", "output_config.effort": "xhigh"},
			absentFields: []string{"thinking.budget_tokens", "thinking.display"},
			response:     claudeStream,
			// Without a summary request the chat client still receives whatever thinking the upstream returned.
			wantReasoning: "Add.",
		},
		{
			name:          "chat format on an anthropic channel keeps a budget for older models",
			channelType:   constant.ChannelTypeAnthropic,
			modelMapping:  `{"gpt-5":"claude-sonnet-4-5"}`,
			check:         modeliqcheck.Check{ReasoningEffort: "medium", MaxTokens: 8192},
			wantPath:      "/v1/messages",
			wantBody:      map[string]any{"max_tokens": 8192.0, "thinking.type": "enabled", "thinking.budget_tokens": 4096.0},
			absentFields:  []string{"output_config"},
			response:      claudeStream,
			wantReasoning: "Add.",
		},
		{
			name:          "responses format requests the summary",
			channelType:   constant.ChannelTypeOpenAI,
			modelMapping:  `{"gpt-5":"gpt-5.2"}`,
			check:         modeliqcheck.Check{EndpointType: "openai-response", ReasoningEffort: "max", ReasoningSummary: true},
			wantPath:      "/v1/responses",
			wantBody:      map[string]any{"model": "gpt-5.2", "reasoning.effort": "xhigh", "reasoning.summary": "auto"},
			response:      responsesStream,
			wantReasoning: "Add.",
		},
		{
			name:          "chat format maps the effort to the model's levels",
			channelType:   constant.ChannelTypeOpenAI,
			check:         modeliqcheck.Check{ReasoningEffort: "none"},
			wantPath:      "/v1/chat/completions",
			wantBody:      map[string]any{"model": "gpt-5", "reasoning_effort": "minimal"},
			response:      chatStream,
			wantReasoning: "Add.",
		},
		{
			name:          "claude format on an openai channel keeps the effort the conversion drops",
			channelType:   constant.ChannelTypeOpenAI,
			modelMapping:  `{"gpt-5":"gpt-5.2"}`,
			check:         modeliqcheck.Check{EndpointType: "anthropic", ReasoningEffort: "high"},
			wantPath:      "/v1/chat/completions",
			wantBody:      map[string]any{"model": "gpt-5.2", "reasoning_effort": "high"},
			absentFields:  []string{"thinking", "output_config", "max_tokens"},
			response:      chatStream,
			wantReasoning: "Add.",
		},
		{
			name:          "gemini format on a gemini channel sets the thinking budget",
			channelType:   constant.ChannelTypeGemini,
			modelMapping:  `{"gpt-5":"gemini-2.5-flash"}`,
			check:         modeliqcheck.Check{EndpointType: "gemini", ReasoningEffort: "low", ReasoningSummary: true},
			wantPath:      "/v1beta/models/gemini-2.5-flash:streamGenerateContent",
			wantBody:      map[string]any{"generationConfig.thinkingConfig.thinkingBudget": 1024.0, "generationConfig.thinkingConfig.includeThoughts": true},
			absentFields:  []string{"generationConfig.thinkingConfig.thinkingLevel"},
			response:      geminiStream,
			wantReasoning: "Add.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotPath string
			var gotBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				gotBody = body
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, test.response)
			}))
			t.Cleanup(server.Close)
			channel := setupModelIQCheckChannel(t, test.channelType, server.URL, test.modelMapping)

			check := test.check
			check.Id, check.Group, check.Model, check.Prompt = "c1", "default", "gpt-5", "2+2?"
			result := model.ModelIQCheckResult{Status: model.ModelIQCheckStatusFailed}
			executeModelIQCheck(context.Background(), check, channel, 1, 600, &result)

			assert.Equal(t, test.wantPath, gotPath)
			require.True(t, gjson.ValidBytes(gotBody), string(gotBody))
			for path, want := range test.wantBody {
				assert.Equal(t, want, gjson.GetBytes(gotBody, path).Value(), "upstream field %s in %s", path, gotBody)
			}
			for _, path := range test.absentFields {
				assert.False(t, gjson.GetBytes(gotBody, path).Exists(), "upstream field %s must be absent in %s", path, gotBody)
			}
			assert.Equal(t, model.ModelIQCheckStatusSuccess, result.Status, result.ErrorMessage)
			assert.Equal(t, "4", result.Content)
			assert.Equal(t, test.wantReasoning, result.Reasoning)
			assert.Equal(t, "stop", result.FinishReason)
		})
	}
}

// An explicit effort cannot be silently dropped for channels whose request
// format the check does not know how to adapt.
func TestExecuteModelIQCheckRejectsEffortForUnadaptableChannels(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the upstream must not be called")
	}))
	t.Cleanup(upstream.Close)
	channel := setupModelIQCheckChannel(t, constant.ChannelTypeCohere, upstream.URL, "")

	result := model.ModelIQCheckResult{Status: model.ModelIQCheckStatusFailed}
	executeModelIQCheck(context.Background(), modeliqcheck.Check{Id: "c1", Group: "default", Model: "gpt-5", Prompt: "2+2?", ReasoningEffort: "high"}, channel, 1, 600, &result)

	assert.Equal(t, model.ModelIQCheckStatusFailed, result.Status)
	assert.Equal(t, modeliqcheck.ErrorTypeUnsupportedModel, result.ErrorType, result.ErrorMessage)
	assert.Contains(t, result.ErrorMessage, "reasoning effort is not supported")
}
