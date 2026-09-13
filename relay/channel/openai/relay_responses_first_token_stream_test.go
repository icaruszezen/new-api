package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupFirstTokenErrorTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.FirstTokenErrorLog{}, &model.FirstTokenErrorBody{}))
	model.DB = db
	model.LOG_DB = db
	prevRedis := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = nil
		model.LOG_DB = nil
		common.RedisEnabled = prevRedis
	})
}

func runResponsesFirstTokenStream(t *testing.T, sendDone bool, events ...string) (*relaycommon.RelayInfo, *dtoUsageResult, *types.NewAPIError) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() {
		constant.StreamingTimeout = oldTimeout
	})

	var body strings.Builder
	for _, event := range events {
		body.WriteString("data: ")
		body.WriteString(event)
		body.WriteString("\n\n")
	}
	if sendDone {
		body.WriteString("data: [DONE]\n\n")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	reqBody := `{"model":"gpt-5.1","input":"hello"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(reqBody)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(common.RequestIdKey, "fte-test-req")
	c.Set("username", "alice")
	c.Set("token_name", "tok")
	c.Set("channel_name", "cpa")
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.1",
		UserId:          7,
		TokenId:         9,
		UsingGroup:      "default",
		IsStream:        true,
		DisablePing:     true,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:         3,
			UpstreamModelName: "custom-test-model",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body.String())),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	usage, apiErr := OaiResponsesStreamHandler(c, info, resp)
	require.NotNil(t, usage)
	return info, &dtoUsageResult{Prompt: usage.PromptTokens, Completion: usage.CompletionTokens, Total: usage.TotalTokens}, apiErr
}

func requireFirstTokenStreamHit(t *testing.T, info *relaycommon.RelayInfo, usage *dtoUsageResult, apiErr *types.NewAPIError) {
	t.Helper()
	require.NotNil(t, apiErr)
	assert.True(t, types.IsSkipRetryError(apiErr))
	assert.True(t, info.FirstTokenErrorHandled)
	assert.Equal(t, 0, usage.Prompt)
	assert.Equal(t, 0, usage.Completion)
	assert.Equal(t, 0, usage.Total)
}

func requireFirstTokenStreamMiss(t *testing.T, info *relaycommon.RelayInfo, apiErr *types.NewAPIError) {
	t.Helper()
	require.Nil(t, apiErr)
	assert.False(t, info.FirstTokenErrorHandled)
}

type dtoUsageResult struct {
	Prompt     int
	Completion int
	Total      int
}

func TestOaiResponsesStreamFirstTokenFailedDoesNotBill(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.output_text.delta","delta":"A"}`,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"cpa fake first token"},"usage":{"input_tokens":88,"output_tokens":1,"total_tokens":89}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, "cpa fake first token", apiErr.Error())
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonFirstTokenError, info.StreamStatus.EndReason)

	var logs []*model.FirstTokenErrorLog
	require.NoError(t, model.DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, 88, logs[0].PromptTokens)
	assert.Equal(t, 1, logs[0].CompletionTokens)
	assert.Equal(t, "cpa fake first token", logs[0].ErrorMessage)
	assert.Equal(t, 3, logs[0].ChannelId)
	assert.Equal(t, "alice", logs[0].Username)

	var errLogs []*model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeError).Find(&errLogs).Error)
	require.Len(t, errLogs, 1)
	assert.Equal(t, "cpa fake first token", errLogs[0].Content)
	assert.Equal(t, 0, errLogs[0].Quota)
	assert.Equal(t, 3, errLogs[0].ChannelId)
}

func TestOaiResponsesStreamFakeCompletedThenFailedKeepsDirtySnapshot(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":17000,"output_tokens":1,"total_tokens":17001}}}`,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"after fake completed"}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, "after fake completed", apiErr.Error())
	var logs []*model.FirstTokenErrorLog
	require.NoError(t, model.DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, 17000, logs[0].PromptTokens)
	assert.Equal(t, 1, logs[0].CompletionTokens)
}

func TestOaiResponsesStreamEofWithoutCompleted(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.output_text.delta","delta":"A"}`,
		`{"type":"response.completed","response":{"status":"in_progress","usage":{"input_tokens":5,"output_tokens":1,"total_tokens":6}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, firstTokenErrorEOFMessage, apiErr.Error())
	var logs []*model.FirstTokenErrorLog
	require.NoError(t, model.DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, 1, logs[0].CompletionTokens)
	assert.Equal(t, firstTokenErrorEOFMessage, logs[0].ErrorMessage)
}

func TestOaiResponsesStreamNormalCompletedNotFirstTokenError(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, true,
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":40,"output_tokens":12,"total_tokens":52}}}`,
	)
	requireFirstTokenStreamMiss(t, info, apiErr)
	assert.Equal(t, 40, usage.Prompt)
	assert.Equal(t, 12, usage.Completion)
	var n int64
	require.NoError(t, model.DB.Model(&model.FirstTokenErrorLog{}).Count(&n).Error)
	assert.Equal(t, int64(0), n)
}

func TestOaiResponsesStreamTrueCompletedOutputOneStrictMode(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	common.FirstTokenErrorTreatAllOutputOneEnabled = false
	info, usage, apiErr := runResponsesFirstTokenStream(t, true,
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":9,"output_tokens":1,"total_tokens":10}}}`,
	)
	requireFirstTokenStreamMiss(t, info, apiErr)
	assert.Equal(t, 1, usage.Completion)
}

func TestOaiResponsesStreamTrueCompletedOutputOneRelaxedMode(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	common.FirstTokenErrorTreatAllOutputOneEnabled = true
	t.Cleanup(func() { common.FirstTokenErrorTreatAllOutputOneEnabled = false })
	info, usage, apiErr := runResponsesFirstTokenStream(t, true,
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":9,"output_tokens":1,"total_tokens":10}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, firstTokenErrorOutputOneMessage, apiErr.Error())
	var logs []*model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeError).Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, firstTokenErrorOutputOneMessage, logs[0].Content)
}

func TestOaiResponsesStreamLogDisabledStillCorrectsBilling(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	common.FirstTokenErrorLogEnabled = false
	t.Cleanup(func() { common.FirstTokenErrorLogEnabled = true })
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"keep correcting"},"usage":{"input_tokens":20,"output_tokens":1}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, "keep correcting", apiErr.Error())
	var n int64
	require.NoError(t, model.DB.Model(&model.FirstTokenErrorLog{}).Count(&n).Error)
	assert.Equal(t, int64(0), n)
	var errLogs []*model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeError).Find(&errLogs).Error)
	require.Len(t, errLogs, 1)
	assert.Equal(t, "keep correcting", errLogs[0].Content)
}

func TestOaiResponsesStreamBodyCaptureAndWindow(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	common.FirstTokenErrorBodyCaptureEnabled = true
	t.Cleanup(func() { common.FirstTokenErrorBodyCaptureEnabled = false })

	for i := 0; i < 11; i++ {
		_, _, apiErr := runResponsesFirstTokenStream(t, false,
			`{"type":"response.failed","response":{"status":"failed","error":{"message":"body window"},"usage":{"input_tokens":2,"output_tokens":1}}}`,
		)
		require.NotNil(t, apiErr)
		assert.True(t, types.IsSkipRetryError(apiErr))
	}
	var logs []*model.FirstTokenErrorLog
	require.NoError(t, model.DB.Order("id asc").Find(&logs).Error)
	require.Len(t, logs, 11)
	var bodies []*model.FirstTokenErrorBody
	require.NoError(t, model.DB.Find(&bodies).Error)
	require.Len(t, bodies, common.FirstTokenErrorBodyKeep)
	var withBody int
	for _, log := range logs {
		if log.HasRequestBody {
			withBody++
		}
	}
	assert.Equal(t, common.FirstTokenErrorBodyKeep, withBody)
}

func TestOaiResponsesStreamFirstTokenErrorReturnsSkipRetry(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"settle zero"},"usage":{"input_tokens":20,"output_tokens":1}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, "settle zero", apiErr.Error())
}

func TestOaiResponsesStreamNormalCompletedDoesNotReturnError(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, true,
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":40,"output_tokens":12,"total_tokens":52}}}`,
	)
	requireFirstTokenStreamMiss(t, info, apiErr)
	assert.Equal(t, 12, usage.Completion)
}

func TestOaiResponsesStreamLogDisabledStillReturnsError(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	common.FirstTokenErrorLogEnabled = false
	t.Cleanup(func() { common.FirstTokenErrorLogEnabled = true })
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"log off settle"},"usage":{"input_tokens":20,"output_tokens":1}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, "log off settle", apiErr.Error())
}

func TestOaiResponsesStreamCorrectionDisabledDoesNotHandle(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	common.FirstTokenErrorCorrectionEnabled = false
	t.Cleanup(func() { common.FirstTokenErrorCorrectionEnabled = true })
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"leave billed"},"usage":{"input_tokens":20,"output_tokens":1}}}`,
	)
	requireFirstTokenStreamMiss(t, info, apiErr)
	assert.Equal(t, 1, usage.Completion)
	var n int64
	require.NoError(t, model.DB.Model(&model.FirstTokenErrorLog{}).Count(&n).Error)
	assert.Equal(t, int64(0), n)
	var errLogs []*model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeError).Find(&errLogs).Error)
	assert.Empty(t, errLogs)
}

func TestOaiResponsesStreamFailedWithoutUsageLongTextDoesNotEstimate(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.output_text.delta","delta":"this is a longer error message that is not a single token"}`,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"no usage"}}}`,
	)
	requireFirstTokenStreamMiss(t, info, apiErr)
	assert.Equal(t, 0, usage.Completion)
	assert.Equal(t, 0, usage.Total)
	var n int64
	require.NoError(t, model.DB.Model(&model.FirstTokenErrorLog{}).Count(&n).Error)
	assert.Equal(t, int64(0), n)
}

func TestOaiResponsesStreamFailedWithoutUsageOneTokenStillHits(t *testing.T) {
	setupFirstTokenErrorTestDB(t)
	info, usage, apiErr := runResponsesFirstTokenStream(t, false,
		`{"type":"response.output_text.delta","delta":"."}`,
		`{"type":"response.failed","response":{"status":"failed","error":{"message":"no usage one"}}}`,
	)
	requireFirstTokenStreamHit(t, info, usage, apiErr)
	assert.Equal(t, "no usage one", apiErr.Error())
	var logs []*model.FirstTokenErrorLog
	require.NoError(t, model.DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, 1, logs[0].CompletionTokens)
}
