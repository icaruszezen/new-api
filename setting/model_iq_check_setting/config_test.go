package model_iq_check_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A check must get at least ten minutes before it counts as timed out, so a
// shorter timeout saved by an earlier version falls back to the default.
func TestGetSettingKeepsRequestTimeoutAtLeastTenMinutes(t *testing.T) {
	saved := config.GlobalConfig.ExportAllConfigs()
	t.Cleanup(func() {
		_ = config.GlobalConfig.LoadFromDB(saved)
	})

	cases := []struct {
		stored string
		want   int
	}{
		{stored: "300", want: DefaultRequestTimeoutSeconds},
		{stored: "599", want: DefaultRequestTimeoutSeconds},
		{stored: "600", want: 600},
		{stored: "1800", want: 1800},
		{stored: "3600", want: 3600},
		{stored: "3601", want: DefaultRequestTimeoutSeconds},
	}
	for _, testCase := range cases {
		require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
			RequestTimeoutOptionKey: testCase.stored,
		}))
		assert.Equal(t, testCase.want, GetSetting().RequestTimeoutSeconds, "stored=%s", testCase.stored)
	}
	assert.GreaterOrEqual(t, DefaultRequestTimeoutSeconds, 600)
}
