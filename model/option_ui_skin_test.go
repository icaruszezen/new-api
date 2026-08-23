package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptionValueGuardsConsoleUISkin(t *testing.T) {
	for _, value := range []string{"classic", "next"} {
		t.Run("accepts "+value, func(t *testing.T) {
			require.NoError(t, validateOptionValue(console_setting.UISkinOptionKey, value))
		})
	}

	for _, value := range []string{"", " ", "Classic", "NEXT", "legacy", "default"} {
		t.Run("rejects "+value, func(t *testing.T) {
			assert.Error(t, validateOptionValue(console_setting.UISkinOptionKey, value))
		})
	}
}
