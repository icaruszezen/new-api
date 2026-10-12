package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type billingCorrectionRuleResponse struct {
	Success bool                        `json:"success"`
	Message string                      `json:"message"`
	Data    model.BillingCorrectionRule `json:"data"`
}

func sendBillingCorrectionRule(t *testing.T, handler gin.HandlerFunc, ruleId int, body map[string]any) (billingCorrectionRuleResponse, string) {
	t.Helper()
	payload, err := common.Marshal(body)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/billing-correction/", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	if ruleId > 0 {
		c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(ruleId)}}
	}

	handler(c)

	var response billingCorrectionRuleResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response, recorder.Body.String()
}

// 上游访问令牌只写不读：接口响应里永远不带令牌，更新时留空沿用已保存的令牌；
// 开启上游核对必须填好用户 ID 和令牌，关闭核对时自动返还一并关闭。
func TestBillingCorrectionRuleUpstreamConfig(t *testing.T) {
	db := openTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.BillingCorrectionRule{}))
	require.NoError(t, db.Create(&model.Channel{Id: 3, Name: "upstream-new-api"}).Error)
	const secret = "upstream-secret-token"
	base := map[string]any{"channel_id": 3, "prompt_tokens_threshold": 1_000_000, "enabled": true}
	with := func(extra map[string]any) map[string]any {
		body := map[string]any{}
		for key, value := range base {
			body[key] = value
		}
		for key, value := range extra {
			body[key] = value
		}
		return body
	}

	rejected := []struct {
		name string
		body map[string]any
	}{
		{name: "missing upstream user id", body: with(map[string]any{"upstream_check_enabled": true, "upstream_access_token": secret})},
		{name: "missing access token", body: with(map[string]any{"upstream_check_enabled": true, "upstream_user_id": 9})},
		{name: "base URL without scheme", body: with(map[string]any{"upstream_check_enabled": true, "upstream_user_id": 9, "upstream_access_token": secret, "upstream_base_url": "relay.example.com"})},
	}
	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := sendBillingCorrectionRule(t, CreateBillingCorrectionRule, 0, tt.body)
			assert.False(t, response.Success)
			assert.NotEmpty(t, response.Message)
		})
	}

	created, raw := sendBillingCorrectionRule(t, CreateBillingCorrectionRule, 0, with(map[string]any{
		"upstream_check_enabled": true,
		"upstream_user_id":       9,
		"upstream_access_token":  secret,
		"upstream_base_url":      "https://relay.example.com/",
		"auto_refund":            true,
	}))
	require.True(t, created.Success, created.Message)
	assert.NotContains(t, raw, secret)
	assert.True(t, created.Data.UpstreamAccessTokenSet)
	assert.Equal(t, "https://relay.example.com", created.Data.UpstreamBaseUrl)
	assert.True(t, created.Data.AutoRefund)
	assert.NotZero(t, created.Data.UpstreamCheckSince)

	// 留空令牌更新：令牌不变；关闭上游核对：自动返还跟着关闭。
	updated, raw := sendBillingCorrectionRule(t, UpdateBillingCorrectionRule, created.Data.Id, with(map[string]any{
		"upstream_check_enabled": false,
		"upstream_user_id":       9,
		"auto_refund":            true,
	}))
	require.True(t, updated.Success, updated.Message)
	assert.NotContains(t, raw, secret)
	assert.False(t, updated.Data.AutoRefund)
	saved, err := model.GetBillingCorrectionRuleById(created.Data.Id)
	require.NoError(t, err)
	assert.Equal(t, secret, saved.UpstreamAccessToken)

	// 重新开启时不填令牌也能用已保存的，并且重新计时。
	saved.UpstreamCheckSince = 1
	require.NoError(t, saved.Update())
	reenabled, _ := sendBillingCorrectionRule(t, UpdateBillingCorrectionRule, created.Data.Id, with(map[string]any{
		"upstream_check_enabled": true,
		"upstream_user_id":       9,
	}))
	require.True(t, reenabled.Success, reenabled.Message)
	assert.Greater(t, reenabled.Data.UpstreamCheckSince, int64(1))
}
