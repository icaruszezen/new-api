package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserQuotaDatesLifetimeReturnsWholeHistorySummary(t *testing.T) {
	setupFlowControllerTestDB(t)
	require.NoError(t, model.DB.Create(&model.QuotaData{
		UserID:       1,
		Username:     "alice",
		ModelName:    "gpt-old",
		CreatedAt:    1,
		TokenUsed:    500,
		PromptTokens: 400,
		CacheTokens:  80,
		Count:        50,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 1)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/data/self?lifetime=1", nil)

	GetUserQuotaDates(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    model.QuotaDataSummary `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success, payload.Message)
	assert.Equal(t, int64(540), payload.Data.TokenUsed)
	assert.Equal(t, int64(400), payload.Data.PromptTokens)
	assert.Equal(t, int64(80), payload.Data.CacheTokens)
	assert.Equal(t, int64(50), payload.Data.CacheSampledCount)
}
