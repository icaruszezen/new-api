package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	channelRetryTestModel     = "channel-retry-model"
	channelRetryTestTokenKey  = "channel-retry-token"
	channelRetryTestQuota     = 1_000_000
	channelRetryTestChannelID = 9

	channelRetryErrorBody   = `{"error":{"message":"upstream is busy","type":"upstream_error","code":"upstream_busy"}}`
	channelRetryChatOK      = `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"channel-retry-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`
	channelRetryResponsesOK = `{"id":"resp_1","object":"response","created_at":1,"status":"completed","model":"channel-retry-model","output":[{"type":"message","id":"msg_1","status":"completed","role":"assistant","content":[{"type":"output_text","text":"ok","annotations":[]}]}],"usage":{"input_tokens":3,"output_tokens":1,"total_tokens":4}}`
	channelRetryClaudeOK    = `{"id":"msg_1","type":"message","role":"assistant","model":"channel-retry-model","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":1}}`
	channelRetryChatSSE     = "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"channel-retry-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n" +
		"data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"channel-retry-model\",\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1,\"total_tokens\":4}}\n\n" +
		"data: [DONE]\n\n"
	channelRetryVideoOK = `{"id":"video_upstream_1","object":"video","model":"channel-retry-model","status":"queued","progress":0,"created_at":1}`

	channelRetryChatRequest      = `{"model":"channel-retry-model","messages":[{"role":"user","content":"hi"}],"x_unknown":1}`
	channelRetryVideoRequest     = `{"model":"channel-retry-model","prompt":"a cat playing piano","seconds":"4"}`
	channelRetryContentTypeJSON  = "application/json"
	channelRetryContentTypeEvent = "text/event-stream"
)

// scriptedUpstream answers the i-th request with statuses[i] (200 once the
// script runs out) and records every request body it receives.
type scriptedUpstream struct {
	mu             sync.Mutex
	statuses       []int
	okContentType  string
	okBody         string
	errContentType string
	bodies         [][]byte
	server         *httptest.Server
}

func newScriptedUpstream(t *testing.T, okContentType string, okBody string, statuses ...int) *scriptedUpstream {
	t.Helper()
	upstream := &scriptedUpstream{statuses: statuses, okContentType: okContentType, okBody: okBody, errContentType: channelRetryContentTypeJSON}
	upstream.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		upstream.mu.Lock()
		attempt := len(upstream.bodies)
		upstream.bodies = append(upstream.bodies, body)
		status := http.StatusOK
		if attempt < len(upstream.statuses) {
			status = upstream.statuses[attempt]
		}
		errContentType := upstream.errContentType
		upstream.mu.Unlock()

		if status != http.StatusOK {
			w.Header().Set("Content-Type", errContentType)
			w.WriteHeader(status)
			_, _ = io.WriteString(w, channelRetryErrorBody)
			return
		}
		w.Header().Set("Content-Type", upstream.okContentType)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, upstream.okBody)
	}))
	t.Cleanup(upstream.server.Close)
	return upstream
}

func (u *scriptedUpstream) requestBodies() [][]byte {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([][]byte(nil), u.bodies...)
}

// setupChannelRetryRelayEnv prepares what Relay()/RelayTask() need after the
// auth and distribute middleware ran: an in-memory database with one user and
// one token holding channelRetryTestQuota, a priced model, and no global retry.
func setupChannelRetryRelayEnv(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.Log{}, &model.UserSubscription{}, &model.Task{}))

	prevRetryTimes := common.RetryTimes
	prevMemoryCache := common.MemoryCacheEnabled
	prevAutoDisable := common.AutomaticDisableChannelEnabled
	prevStreamingTimeout := constant.StreamingTimeout
	prevModelPrices := ratio_setting.ModelPrice2JSONString()
	common.RetryTimes = 0
	common.MemoryCacheEnabled = false
	common.AutomaticDisableChannelEnabled = false
	constant.StreamingTimeout = 300
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"channel-retry-model":0.01}`))
	t.Cleanup(func() {
		common.RetryTimes = prevRetryTimes
		common.MemoryCacheEnabled = prevMemoryCache
		common.AutomaticDisableChannelEnabled = prevAutoDisable
		constant.StreamingTimeout = prevStreamingTimeout
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(prevModelPrices))
	})

	require.NoError(t, db.Create(&model.User{Id: 1, Username: "channel-retry-user", Status: common.UserStatusEnabled, Group: "default", Quota: channelRetryTestQuota}).Error)
	require.NoError(t, db.Create(&model.Token{Id: 1, UserId: 1, Key: channelRetryTestTokenKey, Name: "channel-retry", Status: common.TokenStatusEnabled, RemainQuota: channelRetryTestQuota}).Error)
	return db
}

func createChannelRetryTestChannel(t *testing.T, db *gorm.DB, id int, channelType int, baseURL string, setting string, statusCodeMapping string) *model.Channel {
	t.Helper()
	priority := int64(0)
	weight := uint(1)
	channel := &model.Channel{
		Id:       id,
		Type:     channelType,
		Key:      fmt.Sprintf("sk-channel-retry-%d", id),
		Name:     fmt.Sprintf("channel-retry-%d", id),
		Status:   common.ChannelStatusEnabled,
		Models:   channelRetryTestModel,
		Group:    "default",
		BaseURL:  &baseURL,
		Priority: &priority,
		Weight:   &weight,
	}
	if setting != "" {
		channel.Setting = &setting
	}
	if statusCodeMapping != "" {
		channel.StatusCodeMapping = &statusCodeMapping
	}
	require.NoError(t, db.Create(channel).Error)
	return channel
}

// newChannelRetryRelayContext builds the request context the TokenAuth and
// Distribute middleware leave behind once they selected channel.
func newChannelRetryRelayContext(t *testing.T, path string, channel *model.Channel, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", channelRetryContentTypeJSON)
	common.SetContextKey(c, constant.ContextKeyUserId, 1)
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
	common.SetContextKey(c, constant.ContextKeyTokenId, 1)
	common.SetContextKey(c, constant.ContextKeyTokenKey, channelRetryTestTokenKey)
	common.SetContextKey(c, constant.ContextKeyUserSetting, dto.UserSetting{})
	require.Nil(t, middleware.SetupContextForSelectedChannel(c, channel, channelRetryTestModel))
	return c, recorder
}

func channelRetryConsumeLogs(t *testing.T, db *gorm.DB) []model.Log {
	t.Helper()
	var logs []model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeConsume).Find(&logs).Error)
	return logs
}

func channelRetryAdminUseChannel(t *testing.T, log model.Log) []string {
	t.Helper()
	var other struct {
		AdminInfo struct {
			UseChannel []string `json:"use_channel"`
		} `json:"admin_info"`
	}
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	return other.AdminInfo.UseChannel
}

func channelRetryUserAndTokenQuota(t *testing.T, db *gorm.DB) (int, int) {
	t.Helper()
	var user model.User
	require.NoError(t, db.First(&user, 1).Error)
	var token model.Token
	require.NoError(t, db.First(&token, 1).Error)
	return user.Quota, token.RemainQuota
}

func repeatChannelID(id int, times int) []string {
	chain := make([]string, times)
	for i := range chain {
		chain[i] = fmt.Sprintf("%d", id)
	}
	return chain
}

func TestRelayRetriesSameChannelUntilUpstreamSucceeds(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		format            types.RelayFormat
		channelType       int
		setting           string
		statusCodeMapping string
		globalPassThrough bool
		requestBody       string
		okContentType     string
		okBody            string
		errContentType    string
		statuses          []int
		wantResponse      string
	}{
		{
			name:          "openai chat",
			path:          "/v1/chat/completions",
			format:        types.RelayFormatOpenAI,
			channelType:   constant.ChannelTypeOpenAI,
			setting:       `{"retry_times":1,"retry_interval_ms":0}`,
			requestBody:   channelRetryChatRequest,
			okContentType: channelRetryContentTypeJSON,
			okBody:        channelRetryChatOK,
			statuses:      []int{http.StatusTooManyRequests},
			wantResponse:  `"object":"chat.completion"`,
		},
		{
			name:          "openai chat with channel body pass-through",
			path:          "/v1/chat/completions",
			format:        types.RelayFormatOpenAI,
			channelType:   constant.ChannelTypeOpenAI,
			setting:       `{"retry_times":1,"retry_interval_ms":1,"pass_through_body_enabled":true}`,
			requestBody:   channelRetryChatRequest,
			okContentType: channelRetryContentTypeJSON,
			okBody:        channelRetryChatOK,
			statuses:      []int{http.StatusTooManyRequests},
			wantResponse:  `"object":"chat.completion"`,
		},
		{
			name:              "openai chat with global pass-through",
			path:              "/v1/chat/completions",
			format:            types.RelayFormatOpenAI,
			channelType:       constant.ChannelTypeOpenAI,
			setting:           `{"retry_times":1,"retry_interval_ms":0}`,
			globalPassThrough: true,
			requestBody:       channelRetryChatRequest,
			okContentType:     channelRetryContentTypeJSON,
			okBody:            channelRetryChatOK,
			statuses:          []int{http.StatusTooManyRequests},
			wantResponse:      `"object":"chat.completion"`,
		},
		{
			name:          "openai chat stream",
			path:          "/v1/chat/completions",
			format:        types.RelayFormatOpenAI,
			channelType:   constant.ChannelTypeOpenAI,
			setting:       `{"retry_times":1,"retry_interval_ms":0}`,
			requestBody:   `{"model":"channel-retry-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`,
			okContentType: channelRetryContentTypeEvent,
			okBody:        channelRetryChatSSE,
			statuses:      []int{http.StatusTooManyRequests},
			wantResponse:  `data: [DONE]`,
		},
		{
			name:          "claude messages",
			path:          "/v1/messages",
			format:        types.RelayFormatClaude,
			channelType:   constant.ChannelTypeAnthropic,
			setting:       `{"retry_times":1,"retry_interval_ms":0}`,
			requestBody:   `{"model":"channel-retry-model","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`,
			okContentType: channelRetryContentTypeJSON,
			okBody:        channelRetryClaudeOK,
			statuses:      []int{http.StatusTooManyRequests},
			wantResponse:  `"type":"message"`,
		},
		{
			name:          "openai responses",
			path:          "/v1/responses",
			format:        types.RelayFormatOpenAIResponses,
			channelType:   constant.ChannelTypeOpenAI,
			setting:       `{"retry_times":1,"retry_interval_ms":0}`,
			requestBody:   `{"model":"channel-retry-model","input":"hi"}`,
			okContentType: channelRetryContentTypeJSON,
			okBody:        channelRetryResponsesOK,
			statuses:      []int{http.StatusTooManyRequests},
			wantResponse:  `"object":"response"`,
		},
		{
			name:          "default status list includes 503",
			path:          "/v1/chat/completions",
			format:        types.RelayFormatOpenAI,
			channelType:   constant.ChannelTypeOpenAI,
			setting:       `{"retry_times":1,"retry_interval_ms":0}`,
			requestBody:   channelRetryChatRequest,
			okContentType: channelRetryContentTypeJSON,
			okBody:        channelRetryChatOK,
			statuses:      []int{http.StatusServiceUnavailable},
			wantResponse:  `"object":"chat.completion"`,
		},
		{
			name:              "status code mapping does not hide the upstream status",
			path:              "/v1/chat/completions",
			format:            types.RelayFormatOpenAI,
			channelType:       constant.ChannelTypeOpenAI,
			setting:           `{"retry_times":1,"retry_status_codes":"429","retry_interval_ms":0}`,
			statusCodeMapping: `{"429":"503"}`,
			requestBody:       channelRetryChatRequest,
			okContentType:     channelRetryContentTypeJSON,
			okBody:            channelRetryChatOK,
			statuses:          []int{http.StatusTooManyRequests},
			wantResponse:      `"object":"chat.completion"`,
		},
		{
			name:          "several retries with a status range",
			path:          "/v1/chat/completions",
			format:        types.RelayFormatOpenAI,
			channelType:   constant.ChannelTypeOpenAI,
			setting:       `{"retry_times":3,"retry_status_codes":"429,500-599","retry_interval_ms":0}`,
			requestBody:   channelRetryChatRequest,
			okContentType: channelRetryContentTypeJSON,
			okBody:        channelRetryChatOK,
			statuses:      []int{http.StatusTooManyRequests, http.StatusBadGateway},
			wantResponse:  `"object":"chat.completion"`,
		},
		{
			name:           "non-stream chat after an error answered as an event stream",
			path:           "/v1/chat/completions",
			format:         types.RelayFormatOpenAI,
			channelType:    constant.ChannelTypeOpenAI,
			setting:        `{"retry_times":1,"retry_interval_ms":0}`,
			requestBody:    channelRetryChatRequest,
			okContentType:  channelRetryContentTypeJSON,
			okBody:         channelRetryChatOK,
			errContentType: channelRetryContentTypeEvent,
			statuses:       []int{http.StatusTooManyRequests},
			wantResponse:   `"object":"chat.completion"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupChannelRetryRelayEnv(t)
			globalSettings := model_setting.GetGlobalSettings()
			prevPassThrough := globalSettings.PassThroughRequestEnabled
			globalSettings.PassThroughRequestEnabled = tt.globalPassThrough
			t.Cleanup(func() { globalSettings.PassThroughRequestEnabled = prevPassThrough })

			upstream := newScriptedUpstream(t, tt.okContentType, tt.okBody, tt.statuses...)
			if tt.errContentType != "" {
				upstream.mu.Lock()
				upstream.errContentType = tt.errContentType
				upstream.mu.Unlock()
			}
			channel := createChannelRetryTestChannel(t, db, channelRetryTestChannelID, tt.channelType, upstream.server.URL, tt.setting, tt.statusCodeMapping)
			c, recorder := newChannelRetryRelayContext(t, tt.path, channel, tt.requestBody)

			Relay(c, tt.format)

			wantAttempts := len(tt.statuses) + 1
			wantChain := repeatChannelID(channelRetryTestChannelID, wantAttempts)
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			assert.Contains(t, recorder.Body.String(), tt.wantResponse)
			bodies := upstream.requestBodies()
			require.Len(t, bodies, wantAttempts)
			for i := 1; i < len(bodies); i++ {
				assert.Equal(t, string(bodies[0]), string(bodies[i]), "attempt %d must resend the first attempt's body", i+1)
			}
			assert.Equal(t, wantChain, c.GetStringSlice("use_channel"))

			logs := channelRetryConsumeLogs(t, db)
			require.Len(t, logs, 1, "a retried request is logged once")
			assert.Equal(t, channelRetryTestChannelID, logs[0].ChannelId)
			assert.Equal(t, wantChain, channelRetryAdminUseChannel(t, logs[0]))
			require.Positive(t, logs[0].Quota)
			userQuota, tokenQuota := channelRetryUserAndTokenQuota(t, db)
			assert.Equal(t, channelRetryTestQuota-logs[0].Quota, userQuota, "the user is charged once")
			assert.Equal(t, channelRetryTestQuota-logs[0].Quota, tokenQuota, "the token is charged once")
		})
	}
}

func TestRelayStopsRetryingSameChannel(t *testing.T) {
	tests := []struct {
		name              string
		setting           string
		statusCodeMapping string
		statuses          []int
		wantAttempts      int
		wantStatus        int
	}{
		{
			name:              "retries exhausted returns the mapped status",
			setting:           `{"retry_times":2,"retry_status_codes":"429","retry_interval_ms":0}`,
			statusCodeMapping: `{"429":"503"}`,
			statuses:          []int{http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests},
			wantAttempts:      3,
			wantStatus:        http.StatusServiceUnavailable,
		},
		{
			name:         "status outside the default list",
			setting:      `{"retry_times":2,"retry_interval_ms":0}`,
			statuses:     []int{http.StatusInternalServerError},
			wantAttempts: 1,
			wantStatus:   http.StatusInternalServerError,
		},
		{
			name:         "retry_times unset",
			setting:      `{"retry_status_codes":"429","retry_interval_ms":0}`,
			statuses:     []int{http.StatusTooManyRequests},
			wantAttempts: 1,
			wantStatus:   http.StatusTooManyRequests,
		},
		{
			name:         "no channel settings",
			statuses:     []int{http.StatusTooManyRequests},
			wantAttempts: 1,
			wantStatus:   http.StatusTooManyRequests,
		},
		{
			name:              "mapped status is not matched",
			setting:           `{"retry_times":1,"retry_status_codes":"503","retry_interval_ms":0}`,
			statusCodeMapping: `{"429":"503"}`,
			statuses:          []int{http.StatusTooManyRequests},
			wantAttempts:      1,
			wantStatus:        http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupChannelRetryRelayEnv(t)
			upstream := newScriptedUpstream(t, channelRetryContentTypeJSON, channelRetryChatOK, tt.statuses...)
			channel := createChannelRetryTestChannel(t, db, channelRetryTestChannelID, constant.ChannelTypeOpenAI, upstream.server.URL, tt.setting, tt.statusCodeMapping)
			c, recorder := newChannelRetryRelayContext(t, "/v1/chat/completions", channel, channelRetryChatRequest)

			Relay(c, types.RelayFormatOpenAI)

			assert.Equal(t, tt.wantStatus, recorder.Code, recorder.Body.String())
			assert.Len(t, upstream.requestBodies(), tt.wantAttempts)
			assert.Equal(t, repeatChannelID(channelRetryTestChannelID, tt.wantAttempts), c.GetStringSlice("use_channel"))
			assert.Empty(t, channelRetryConsumeLogs(t, db))
			// The pre-consumed quota is returned asynchronously.
			require.Eventually(t, func() bool {
				userQuota, tokenQuota := channelRetryUserAndTokenQuota(t, db)
				return userQuota == channelRetryTestQuota && tokenQuota == channelRetryTestQuota
			}, 5*time.Second, 10*time.Millisecond, "the pre-consumed quota must be refunded")
		})
	}
}

func TestRelayChannelRetriesDoNotConsumeGlobalRetries(t *testing.T) {
	db := setupChannelRetryRelayEnv(t)
	common.RetryTimes = 1

	first := newScriptedUpstream(t, channelRetryContentTypeJSON, channelRetryChatOK, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests)
	second := newScriptedUpstream(t, channelRetryContentTypeJSON, channelRetryChatOK)
	firstChannel := createChannelRetryTestChannel(t, db, 7, constant.ChannelTypeOpenAI, first.server.URL, `{"retry_times":1,"retry_interval_ms":0}`, "")
	createChannelRetryTestChannel(t, db, 8, constant.ChannelTypeOpenAI, second.server.URL, "", "")
	priority := int64(0)
	// Only channel 8 is selectable for the global retry, so the pick is deterministic.
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: channelRetryTestModel, ChannelId: 7, Enabled: false, Priority: &priority, Weight: 1}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: channelRetryTestModel, ChannelId: 8, Enabled: true, Priority: &priority, Weight: 1}).Error)
	c, recorder := newChannelRetryRelayContext(t, "/v1/chat/completions", firstChannel, channelRetryChatRequest)

	Relay(c, types.RelayFormatOpenAI)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assert.Len(t, first.requestBodies(), 2, "channel 7 is retried once on its own")
	assert.Len(t, second.requestBodies(), 1, "the global retry still switches to channel 8")
	wantChain := []string{"7", "7", "8"}
	assert.Equal(t, wantChain, c.GetStringSlice("use_channel"))
	logs := channelRetryConsumeLogs(t, db)
	require.Len(t, logs, 1)
	assert.Equal(t, 8, logs[0].ChannelId)
	assert.Equal(t, wantChain, channelRetryAdminUseChannel(t, logs[0]))
}

func TestRelayTaskRetriesSameChannel(t *testing.T) {
	tests := []struct {
		name         string
		setting      string
		statuses     []int
		wantAttempts int
		wantStatus   int
	}{
		{
			name:         "retry succeeds",
			setting:      `{"retry_times":1,"retry_interval_ms":0}`,
			statuses:     []int{http.StatusTooManyRequests},
			wantAttempts: 2,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "retries exhausted",
			setting:      `{"retry_times":1,"retry_interval_ms":0}`,
			statuses:     []int{http.StatusTooManyRequests, http.StatusTooManyRequests},
			wantAttempts: 2,
			wantStatus:   http.StatusTooManyRequests,
		},
		{
			name:         "status outside the list",
			setting:      `{"retry_times":1,"retry_interval_ms":0}`,
			statuses:     []int{http.StatusInternalServerError},
			wantAttempts: 1,
			wantStatus:   http.StatusInternalServerError,
		},
		{
			name:         "retry not configured",
			statuses:     []int{http.StatusTooManyRequests},
			wantAttempts: 1,
			wantStatus:   http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupChannelRetryRelayEnv(t)
			upstream := newScriptedUpstream(t, channelRetryContentTypeJSON, channelRetryVideoOK, tt.statuses...)
			// An OpenAI channel serves /v1/videos through the sora task adaptor.
			channel := createChannelRetryTestChannel(t, db, channelRetryTestChannelID, constant.ChannelTypeOpenAI, upstream.server.URL, tt.setting, "")
			c, recorder := newChannelRetryRelayContext(t, "/v1/videos", channel, channelRetryVideoRequest)
			c.Set("relay_mode", relayconstant.RelayModeVideoSubmit)

			RelayTask(c)

			require.Equal(t, tt.wantStatus, recorder.Code, recorder.Body.String())
			bodies := upstream.requestBodies()
			require.Len(t, bodies, tt.wantAttempts)
			for i := 1; i < len(bodies); i++ {
				assert.Equal(t, string(bodies[0]), string(bodies[i]), "attempt %d must resend the first attempt's body", i+1)
			}
			assert.Equal(t, repeatChannelID(channelRetryTestChannelID, tt.wantAttempts), c.GetStringSlice("use_channel"))

			var tasks []model.Task
			require.NoError(t, db.Find(&tasks).Error)
			logs := channelRetryConsumeLogs(t, db)
			if tt.wantStatus != http.StatusOK {
				assert.Empty(t, tasks)
				assert.Empty(t, logs)
				require.Eventually(t, func() bool {
					userQuota, tokenQuota := channelRetryUserAndTokenQuota(t, db)
					return userQuota == channelRetryTestQuota && tokenQuota == channelRetryTestQuota
				}, 5*time.Second, 10*time.Millisecond, "the pre-consumed quota must be refunded")
				return
			}
			require.Len(t, tasks, 1, "a retried submit creates one task")
			assert.Equal(t, "video_upstream_1", tasks[0].PrivateData.UpstreamTaskID)
			require.Len(t, logs, 1, "a retried submit is logged once")
			require.Positive(t, logs[0].Quota)
			userQuota, tokenQuota := channelRetryUserAndTokenQuota(t, db)
			assert.Equal(t, channelRetryTestQuota-logs[0].Quota, userQuota, "the user is charged once")
			assert.Equal(t, channelRetryTestQuota-logs[0].Quota, tokenQuota, "the token is charged once")
		})
	}
}

// channelRetryDecisionContext returns a request context whose selected channel
// has the given retry settings, and the policy Relay() derives from it.
func channelRetryDecisionContext(t *testing.T, settings dto.ChannelSettings) (*gin.Context, service.ChannelRetryPolicy) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(nil))
	common.SetContextKey(c, constant.ContextKeyChannelSetting, settings)
	return c, service.GetChannelRetryPolicy(c)
}

// useAutomaticDisableRules sets the global automatic channel-disable rules for
// one test and restores them afterwards.
func useAutomaticDisableRules(t *testing.T, statusCodes string, keyword string) {
	t.Helper()
	prevEnabled := common.AutomaticDisableChannelEnabled
	prevStatusCodes := operation_setting.AutomaticDisableStatusCodesToString()
	prevKeywords := operation_setting.AutomaticDisableKeywordsToString()
	require.NoError(t, operation_setting.AutomaticDisableStatusCodesFromString(statusCodes))
	operation_setting.AutomaticDisableKeywordsFromString(keyword)
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = prevEnabled
		require.NoError(t, operation_setting.AutomaticDisableStatusCodesFromString(prevStatusCodes))
		operation_setting.AutomaticDisableKeywordsFromString(prevKeywords)
	})
}

func TestShouldRetrySameChannel(t *testing.T) {
	useAutomaticDisableRules(t, "401", "account suspended")

	noWait := 0
	retry429And503 := dto.ChannelSettings{RetryTimes: 2, RetryStatusCodes: "429,503", RetryIntervalMs: &noWait}
	upstreamError := func(status int, upstreamStatus int, message string, opts ...types.NewAPIErrorOptions) *types.NewAPIError {
		opts = append(opts, types.ErrOptionWithUpstreamStatusCode(upstreamStatus))
		return types.NewOpenAIError(errors.New(message), types.ErrorCodeBadResponseStatusCode, status, opts...)
	}

	tests := []struct {
		name          string
		settings      dto.ChannelSettings
		format        types.RelayFormat
		autoBan       bool
		autoDisable   bool
		writeResponse bool
		cancelRequest bool
		err           *types.NewAPIError
		retried       int
		want          bool
	}{
		{
			name:     "upstream status in the list",
			settings: retry429And503,
			err:      upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "rate limited"),
			want:     true,
		},
		{
			name:     "last allowed retry",
			settings: retry429And503,
			err:      upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "rate limited"),
			retried:  1,
			want:     true,
		},
		{
			name:     "retries exhausted",
			settings: retry429And503,
			err:      upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "rate limited"),
			retried:  2,
			want:     false,
		},
		{
			name:     "retry not configured",
			settings: dto.ChannelSettings{RetryStatusCodes: "429"},
			err:      upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "rate limited"),
			want:     false,
		},
		{
			name:     "nil error",
			settings: retry429And503,
			want:     false,
		},
		{
			name:     "upstream status not in the list",
			settings: retry429And503,
			err:      upstreamError(http.StatusInternalServerError, http.StatusInternalServerError, "boom"),
			want:     false,
		},
		{
			name:     "matches the upstream status before mapping",
			settings: retry429And503,
			err:      upstreamError(http.StatusBadGateway, http.StatusTooManyRequests, "rate limited"),
			want:     true,
		},
		{
			name:     "mapped status alone does not match",
			settings: dto.ChannelSettings{RetryTimes: 1, RetryStatusCodes: "502", RetryIntervalMs: &noWait},
			err:      upstreamError(http.StatusBadGateway, http.StatusTooManyRequests, "rate limited"),
			want:     false,
		},
		{
			name:     "local error with a listed status",
			settings: retry429And503,
			err:      types.NewOpenAIError(errors.New("local limit"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests),
			want:     false,
		},
		{
			name:     "skip-retry error",
			settings: retry429And503,
			err:      upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "rate limited", types.ErrOptionWithSkipRetry()),
			want:     false,
		},
		{
			name:     "channel error",
			settings: retry429And503,
			err:      types.NewError(errors.New("no key"), types.ErrorCodeChannelNoAvailableKey, types.ErrOptionWithUpstreamStatusCode(http.StatusTooManyRequests)),
			want:     false,
		},
		{
			name:          "response already written",
			settings:      retry429And503,
			writeResponse: true,
			err:           upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "rate limited"),
			want:          false,
		},
		{
			name:          "realtime ignores the hijacked writer",
			settings:      retry429And503,
			format:        types.RelayFormatOpenAIRealtime,
			writeResponse: true,
			err:           upstreamError(http.StatusInternalServerError, http.StatusTooManyRequests, "rate limited"),
			want:          true,
		},
		{
			name:          "client went away",
			settings:      retry429And503,
			cancelRequest: true,
			err:           upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "rate limited"),
			want:          false,
		},
		{
			name:        "auto-ban channel about to be disabled by keyword",
			settings:    retry429And503,
			autoBan:     true,
			autoDisable: true,
			err:         upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "account suspended"),
			want:        false,
		},
		{
			name:        "auto-ban channel about to be disabled by status",
			settings:    dto.ChannelSettings{RetryTimes: 1, RetryStatusCodes: "401", RetryIntervalMs: &noWait},
			autoBan:     true,
			autoDisable: true,
			err:         upstreamError(http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized"),
			want:        false,
		},
		{
			name:        "disable keyword without auto-ban still retries",
			settings:    retry429And503,
			autoDisable: true,
			err:         upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "account suspended"),
			want:        true,
		},
		{
			name:     "auto-ban with automatic disabling off still retries",
			settings: retry429And503,
			autoBan:  true,
			err:      upstreamError(http.StatusTooManyRequests, http.StatusTooManyRequests, "account suspended"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.AutomaticDisableChannelEnabled = tt.autoDisable
			c, policy := channelRetryDecisionContext(t, tt.settings)
			if tt.writeResponse {
				c.Writer.WriteHeaderNow()
			}
			if tt.cancelRequest {
				ctx, cancel := context.WithCancel(c.Request.Context())
				cancel()
				c.Request = c.Request.WithContext(ctx)
			}
			format := tt.format
			if format == "" {
				format = types.RelayFormatOpenAI
			}

			assert.Equal(t, tt.want, shouldRetrySameChannel(c, format, policy, tt.autoBan, tt.err, tt.retried))
		})
	}
}

func TestShouldRetryTaskSameChannel(t *testing.T) {
	useAutomaticDisableRules(t, "401", "account suspended")

	noWait := 0
	settings := dto.ChannelSettings{RetryTimes: 1, RetryIntervalMs: &noWait}
	upstreamTaskError := func(status int, message string) *taskdto.TaskError {
		taskErr := service.TaskErrorWrapper(errors.New(message), "fail_to_fetch_task", status)
		taskErr.UpstreamStatusCode = status
		return taskErr
	}

	tests := []struct {
		name          string
		autoBan       bool
		autoDisable   bool
		writeResponse bool
		cancelRequest bool
		err           *taskdto.TaskError
		retried       int
		want          bool
	}{
		{name: "upstream status in the default list", err: upstreamTaskError(http.StatusServiceUnavailable, "busy"), want: true},
		{name: "nil error", want: false},
		{name: "retries exhausted", err: upstreamTaskError(http.StatusTooManyRequests, "busy"), retried: 1, want: false},
		{name: "upstream status not in the list", err: upstreamTaskError(http.StatusInternalServerError, "boom"), want: false},
		{name: "error without upstream status", err: service.TaskErrorWrapper(errors.New("dial failed"), "do_request_failed", http.StatusTooManyRequests), want: false},
		{name: "response already written", writeResponse: true, err: upstreamTaskError(http.StatusTooManyRequests, "busy"), want: false},
		{name: "client went away", cancelRequest: true, err: upstreamTaskError(http.StatusTooManyRequests, "busy"), want: false},
		{name: "auto-ban channel about to be disabled", autoBan: true, autoDisable: true, err: upstreamTaskError(http.StatusTooManyRequests, "account suspended"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.AutomaticDisableChannelEnabled = tt.autoDisable
			c, policy := channelRetryDecisionContext(t, settings)
			if tt.writeResponse {
				c.Writer.WriteHeaderNow()
			}
			if tt.cancelRequest {
				ctx, cancel := context.WithCancel(c.Request.Context())
				cancel()
				c.Request = c.Request.WithContext(ctx)
			}

			assert.Equal(t, tt.want, shouldRetryTaskSameChannel(c, policy, tt.autoBan, tt.err, tt.retried))
		})
	}
}

func TestShouldRetryStopsOnceClientIsGone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newContext := func(cancelRequest bool) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(ctx)
		if cancelRequest {
			cancel()
		}
		return c
	}
	upstreamFailure := types.NewOpenAIError(errors.New("bad response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	channelFailure := types.NewError(errors.New("no key"), types.ErrorCodeChannelNoAvailableKey)

	assert.True(t, shouldRetry(newContext(false), upstreamFailure, 1), "a connected client gets another channel")
	// After the client disconnected, a retry would be read to the end and
	// billed on a channel that drains, with nobody waiting for the answer.
	assert.False(t, shouldRetry(newContext(true), upstreamFailure, 1))
	assert.False(t, shouldRetry(newContext(true), channelFailure, 1))
}
