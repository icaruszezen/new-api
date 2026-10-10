package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

func TestFormatUserLogsStripsLegacyPrivilegedFields(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"request_path":  "/v1/chat/completions",
		"channel_id":    202,
		"channel_name":  "legacy-secret-channel",
		"channel_type":  1,
		"reject_reason": "legacy-policy-rejection",
		"admin_info": map[string]interface{}{
			"existing_admin_field": "preserved",
		},
		"audit_info": map[string]interface{}{
			"method": "POST",
		},
	})
	logs := []*Log{{
		Id:          99,
		ChannelId:   77,
		ChannelName: "resolved-secret-channel",
		Other:       other,
	}}

	formatUserLogs(logs, 10)

	assert.Equal(t, 11, logs[0].Id)
	assert.Equal(t, 77, logs[0].ChannelId)
	assert.Empty(t, logs[0].ChannelName)
	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.Equal(t, "/v1/chat/completions", parsed["request_path"])
	for _, key := range []string{
		"channel_id",
		"channel_name",
		"channel_type",
		"reject_reason",
		"admin_info",
		"audit_info",
	} {
		assert.NotContains(t, parsed, key)
	}
}
