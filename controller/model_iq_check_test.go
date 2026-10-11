package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModelIQCheckConfigTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Log{}))

	previousOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		common.OptionMap = previousOptionMap
		_ = config.GlobalConfig.LoadFromDB(saved)
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		model_iq_check_setting.EnabledOptionKey: "false",
		model_iq_check_setting.URLOptionKey:     "",
	}))
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

func storedOptionValue(t *testing.T, db *gorm.DB, key string) string {
	t.Helper()
	var option model.Option
	require.NoError(t, db.Where(&model.Option{Key: key}).First(&option).Error)
	return option.Value
}

// The status API publishes the entry only while it is switched on, and
// switching it off keeps the address so the admin can turn it back on.
func TestUpdateModelIQCheckConfigPublishesEntryURL(t *testing.T) {
	db := setupModelIQCheckConfigTest(t)

	success, message := putModelIQCheckConfig(t, map[string]any{
		"enabled": true,
		"url":     "  https://iq.example.com/check?from=new-api  ",
	})
	require.True(t, success, message)
	assert.Equal(t, "https://iq.example.com/check?from=new-api", model_iq_check_setting.EntryURL())
	assert.Equal(t, "true", storedOptionValue(t, db, model_iq_check_setting.EnabledOptionKey))
	assert.Equal(t, "https://iq.example.com/check?from=new-api", storedOptionValue(t, db, model_iq_check_setting.URLOptionKey))

	success, message = putModelIQCheckConfig(t, map[string]any{
		"enabled": false,
		"url":     "https://iq.example.com/check?from=new-api",
	})
	require.True(t, success, message)
	assert.Empty(t, model_iq_check_setting.EntryURL(), "a disabled entry is hidden")
	assert.Equal(t, "https://iq.example.com/check?from=new-api", model_iq_check_setting.GetSetting().URL)
	assert.Equal(t, "false", storedOptionValue(t, db, model_iq_check_setting.EnabledOptionKey))
}

// The address is rendered as a link for every visitor, so anything other than
// a full http(s) URL is rejected and the saved entry stays as it was.
func TestUpdateModelIQCheckConfigRejectsUnsafeURLWithoutSaving(t *testing.T) {
	setupModelIQCheckConfigTest(t)
	success, message := putModelIQCheckConfig(t, map[string]any{"enabled": true, "url": "https://iq.example.com"})
	require.True(t, success, message)

	cases := []struct {
		name    string
		enabled bool
		url     string
	}{
		{name: "javascript scheme", enabled: true, url: "javascript:alert(1)"},
		{name: "data scheme", enabled: true, url: "data:text/html,<script>alert(1)</script>"},
		{name: "ftp scheme", enabled: true, url: "ftp://iq.example.com"},
		{name: "relative path", enabled: true, url: "/iq-check"},
		{name: "protocol relative", enabled: true, url: "//iq.example.com"},
		{name: "missing host", enabled: true, url: "https://"},
		{name: "too long", enabled: true, url: "https://iq.example.com/" + strings.Repeat("a", model_iq_check_setting.MaxURLLength)},
		{name: "enabled without url", enabled: true, url: "   "},
		{name: "unsafe url while disabled", enabled: false, url: "javascript:alert(1)"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			success, message := putModelIQCheckConfig(t, map[string]any{"enabled": testCase.enabled, "url": testCase.url})
			assert.False(t, success)
			assert.NotEmpty(t, message)
			assert.Equal(t, model_iq_check_setting.ModelIQCheckSetting{Enabled: true, URL: "https://iq.example.com"}, model_iq_check_setting.GetSetting())
		})
	}
}
