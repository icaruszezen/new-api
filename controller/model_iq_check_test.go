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
	require.NotNil(t, claudeReq.MaxTokens, "the Anthropic API requires max_tokens, so it is never dropped")
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

func iqCheckChunk(content string) string {
	return `data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-5","choices":[{"index":0,"delta":{"content":` + strconv.Quote(content) + `}}]}` + "\n\n"
}

// A reasoning model can stay silent far longer than STREAMING_TIMEOUT before it
// answers; the check must keep waiting until its own timeout.
func TestExecuteModelIQCheckOutlastsStreamingIdleTimeout(t *testing.T) {
	previousStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() { constant.StreamingTimeout = previousStreamingTimeout })

	var streamed atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		streamed.Store(strings.Contains(string(body), `"stream":true`))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, iqCheckChunk(""))
		w.(http.Flusher).Flush()
		time.Sleep(2 * time.Second)
		_, _ = io.WriteString(w, iqCheckChunk("4"))
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(upstream.Close)
	channel := setupModelIQCheckChannel(t, constant.ChannelTypeOpenAI, upstream.URL, "")

	result := model.ModelIQCheckResult{Status: model.ModelIQCheckStatusFailed}
	executeModelIQCheck(context.Background(), modeliqcheck.Check{Id: "c1", Group: "default", Model: "gpt-5", Prompt: "2+2?"}, channel, 1, 600, &result)

	assert.True(t, streamed.Load(), "the upstream request must ask for a stream")
	assert.Equal(t, model.ModelIQCheckStatusSuccess, result.Status, result.ErrorMessage)
	assert.Equal(t, "4", result.Content)
}

// A stream cut before it finishes is a failed check that keeps the partial
// answer, not a success with half an answer.
func TestExecuteModelIQCheckFailsWhenUpstreamStreamBreaks(t *testing.T) {
	previousStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 300
	t.Cleanup(func() { constant.StreamingTimeout = previousStreamingTimeout })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, buffered, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		defer conn.Close()
		chunk := iqCheckChunk("partial answer")
		_, _ = io.WriteString(buffered, "HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n")
		_, _ = io.WriteString(buffered, strconv.FormatInt(int64(len(chunk)), 16)+"\r\n"+chunk+"\r\n")
		_ = buffered.Flush()
	}))
	t.Cleanup(upstream.Close)
	channel := setupModelIQCheckChannel(t, constant.ChannelTypeOpenAI, upstream.URL, "")

	result := model.ModelIQCheckResult{Status: model.ModelIQCheckStatusFailed}
	executeModelIQCheck(context.Background(), modeliqcheck.Check{Id: "c1", Group: "default", Model: "gpt-5", Prompt: "2+2?"}, channel, 1, 600, &result)

	assert.Equal(t, model.ModelIQCheckStatusFailed, result.Status)
	assert.Equal(t, modeliqcheck.ErrorTypeUpstream, result.ErrorType)
	assert.Contains(t, result.ErrorMessage, "stream ended before completion")
	assert.Equal(t, "partial answer", result.Content)
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

// Some channels write through gin's c.Stream, which needs a CloseNotifier; the
// test recorder must provide one instead of panicking, and stop the stream
// once the check's context ends.
func TestChannelTestResponseWriterSupportsGinStream(t *testing.T) {
	recorder := httptest.NewRecorder()
	clientGone := make(chan bool)
	c, _ := gin.CreateTestContext(channelTestResponseWriter{ResponseRecorder: recorder, closed: clientGone})

	steps := 0
	assert.False(t, c.Stream(func(w io.Writer) bool {
		steps++
		_, _ = io.WriteString(w, "data: x\n\n")
		return steps < 2
	}))
	assert.Equal(t, "data: x\n\ndata: x\n\n", recorder.Body.String())

	close(clientGone)
	assert.True(t, c.Stream(func(io.Writer) bool {
		t.Error("no step may run after the client is gone")
		return false
	}))
}
