package model

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetQuotaDataSummaryByUserIdAggregatesWholeHistoryForOneUser(t *testing.T) {
	truncateTables(t)

	// 两个桶相隔远超看板接口的 30 天窗口，累计值必须同时包含它们。
	require.NoError(t, DB.Create(&QuotaData{
		UserID:       1,
		Username:     "alice",
		ModelName:    "gpt-a",
		CreatedAt:    1_000_000,
		TokenUsed:    300,
		PromptTokens: 200,
		CacheTokens:  50,
		Count:        2,
		Quota:        100,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID:       1,
		Username:     "alice",
		ModelName:    "gpt-b",
		CreatedAt:    900_000_000,
		TokenUsed:    120,
		PromptTokens: 100,
		CacheTokens:  30,
		Count:        1,
		Quota:        40,
	}).Error)
	// 缓存列上线前写的桶：有请求数但没有输入 token，不能进缓存读取率的样本。
	require.NoError(t, DB.Create(&QuotaData{
		UserID:    1,
		Username:  "alice",
		ModelName: "gpt-legacy",
		CreatedAt: 800_000,
		TokenUsed: 700,
		Count:     40,
		Quota:     60,
	}).Error)
	require.NoError(t, DB.Create(&QuotaData{
		UserID:       2,
		Username:     "bob",
		ModelName:    "gpt-a",
		CreatedAt:    1_000_000,
		TokenUsed:    999,
		PromptTokens: 999,
		CacheTokens:  999,
		Count:        5,
		Quota:        500,
	}).Error)

	summary, err := GetQuotaDataSummaryByUserId(1)

	require.NoError(t, err)
	assert.Equal(t, int64(1120), summary.TokenUsed)
	assert.Equal(t, int64(300), summary.PromptTokens)
	assert.Equal(t, int64(80), summary.CacheTokens)
	assert.Equal(t, int64(3), summary.CacheSampledCount)
}

func TestGetQuotaDataSummaryByUserIdReturnsZerosWithoutRows(t *testing.T) {
	truncateTables(t)

	summary, err := GetQuotaDataSummaryByUserId(42)

	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.TokenUsed)
	assert.Equal(t, int64(0), summary.PromptTokens)
	assert.Equal(t, int64(0), summary.CacheTokens)
	assert.Equal(t, int64(0), summary.CacheSampledCount)
}

func TestLogQuotaDataAccumulatesPromptAndCacheTokensInOneHourBucket(t *testing.T) {
	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()
	t.Cleanup(func() {
		CacheQuotaDataLock.Lock()
		CacheQuotaData = make(map[string]*QuotaData)
		CacheQuotaDataLock.Unlock()
	})

	params := QuotaDataLogParams{
		UserID:       7,
		Username:     "carol",
		ModelName:    "gpt-a",
		Quota:        10,
		CreatedAt:    3600,
		TokenUsed:    90,
		PromptTokens: 60,
		CacheTokens:  25,
	}
	LogQuotaData(params)
	SaveQuotaDataCache()

	// 第二轮落库走 increaseQuotaData 的 UPDATE 分支，新列必须累加而不是被覆盖。
	LogQuotaData(params)
	SaveQuotaDataCache()

	summary, err := GetQuotaDataSummaryByUserId(7)
	require.NoError(t, err)
	assert.Equal(t, int64(180), summary.TokenUsed)
	assert.Equal(t, int64(120), summary.PromptTokens)
	assert.Equal(t, int64(50), summary.CacheTokens)
	assert.Equal(t, int64(2), summary.CacheSampledCount)

	var rows int64
	require.NoError(t, DB.Model(&QuotaData{}).Where("user_id = ?", 7).Count(&rows).Error)
	assert.Equal(t, int64(1), rows)
}

func TestRecordConsumeLogCarriesCacheTokensIntoQuotaData(t *testing.T) {
	cases := []struct {
		name        string
		cacheTokens any
		expected    int64
	}{
		{name: "int from the relay billing path", cacheTokens: 400, expected: 400},
		{name: "float64 from a JSON round trip", cacheTokens: float64(400), expected: 400},
		{name: "absent for platforms without prompt caching", cacheTokens: nil, expected: 0},
		{name: "negative upstream value is discarded", cacheTokens: -5, expected: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			previousExport := common.DataExportEnabled
			common.DataExportEnabled = true
			CacheQuotaDataLock.Lock()
			CacheQuotaData = make(map[string]*QuotaData)
			CacheQuotaDataLock.Unlock()
			t.Cleanup(func() {
				common.DataExportEnabled = previousExport
				CacheQuotaDataLock.Lock()
				CacheQuotaData = make(map[string]*QuotaData)
				CacheQuotaDataLock.Unlock()
			})

			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
			ctx.Set("username", "dana")

			other := map[string]interface{}{"model_ratio": 1.0}
			if tc.cacheTokens != nil {
				other["cache_tokens"] = tc.cacheTokens
			}
			RecordConsumeLog(ctx, 21, RecordConsumeLogParams{
				PromptTokens:     1000,
				CompletionTokens: 200,
				ModelName:        "gpt-a",
				Quota:            30,
				Other:            other,
			})
			SaveQuotaDataCache()

			summary, err := GetQuotaDataSummaryByUserId(21)
			require.NoError(t, err)
			assert.Equal(t, int64(1200), summary.TokenUsed)
			assert.Equal(t, int64(1000), summary.PromptTokens)
			assert.Equal(t, tc.expected, summary.CacheTokens)
		})
	}
}
