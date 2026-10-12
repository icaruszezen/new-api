package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The IQ check address is rendered as a link for every visitor, so the generic
// option API must reject the same unsafe values as the dedicated endpoint.
func TestValidateOptionValueGuardsModelIQCheckURL(t *testing.T) {
	for _, value := range []string{"", "https://iq.example.com", "http://iq.example.com:8080/check?from=new-api"} {
		t.Run("accepts "+value, func(t *testing.T) {
			require.NoError(t, validateOptionValue(model_iq_check_setting.URLOptionKey, value))
		})
	}

	rejected := []struct {
		name  string
		value string
	}{
		{name: "javascript scheme", value: "javascript:alert(1)"},
		{name: "data scheme", value: "data:text/html,<script>alert(1)</script>"},
		{name: "ftp scheme", value: "ftp://iq.example.com"},
		{name: "relative path", value: "/iq-check"},
		{name: "protocol relative", value: "//iq.example.com"},
		{name: "missing host", value: "https://"},
		{name: "leading space", value: " https://iq.example.com"},
		{name: "too long", value: "https://iq.example.com/" + strings.Repeat("a", model_iq_check_setting.MaxURLLength)},
	}
	for _, testCase := range rejected {
		t.Run("rejects "+testCase.name, func(t *testing.T) {
			assert.Error(t, validateOptionValue(model_iq_check_setting.URLOptionKey, testCase.value))
		})
	}
}
