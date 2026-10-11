package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// go-epay v0.0.5 refuses to sign any value containing '&', so a plan title
// with '&' must still produce a signable order instead of an expired one.
func TestSubscriptionRequestEpayAcceptsAmpersandInPlanTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	confirmPaymentComplianceForTest(t)
	previousDB := model.DB
	previousType := common.MainDatabaseType()
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalGateways := operation_setting.EpayGateways
	originalServerAddress := system_setting.ServerAddress
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		operation_setting.EpayGateways = originalGateways
		system_setting.ServerAddress = originalServerAddress
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SubscriptionPlan{}, &model.SubscriptionOrder{}))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "epay_id"
	operation_setting.EpayKey = "test-key"
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}}
	operation_setting.EpayGateways = nil
	system_setting.ServerAddress = "https://dashboard.example.com"

	plan := &model.SubscriptionPlan{Title: "Pro & Team", PriceAmount: 9.9, Enabled: true}
	require.NoError(t, db.Create(plan).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := fmt.Sprintf(`{"plan_id":%d,"payment_method":"alipay"}`, plan.Id)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/subscription/epay/pay", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 7)

	SubscriptionRequestEpay(c)

	var result struct {
		Message string            `json:"message"`
		Data    map[string]string `json:"data"`
	}
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result), recorder.Body.String())
	require.Equal(t, "success", result.Message, recorder.Body.String())
	assert.Equal(t, "SUB:Pro ＆ Team", result.Data["name"])
	assert.NotEmpty(t, result.Data["sign"])

	var order model.SubscriptionOrder
	require.NoError(t, db.Where("plan_id = ?", plan.Id).First(&order).Error)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
}
