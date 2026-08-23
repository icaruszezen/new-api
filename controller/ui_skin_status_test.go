package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatusNormalizesUISkin(t *testing.T) {
	cases := []struct {
		name     string
		stored   string
		expected string
	}{
		{name: "unconfigured value falls back to classic", stored: "", expected: "classic"},
		{name: "classic is advertised as classic", stored: "classic", expected: "classic"},
		{name: "next is advertised as next", stored: "next", expected: "next"},
		{name: "invalid value falls back to classic", stored: "legacy", expected: "classic"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			previousMap := common.OptionMap
			common.OptionMap = map[string]string{}
			t.Cleanup(func() { common.OptionMap = previousMap })

			consoleSetting := console_setting.GetConsoleSetting()
			previousSkin := consoleSetting.UISkin
			consoleSetting.UISkin = testCase.stored
			t.Cleanup(func() { consoleSetting.UISkin = previousSkin })

			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)

			GetStatus(context)

			var payload struct {
				Success bool           `json:"success"`
				Data    map[string]any `json:"data"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.True(t, payload.Success)
			assert.Equal(t, testCase.expected, payload.Data["ui_skin"])
		})
	}
}
