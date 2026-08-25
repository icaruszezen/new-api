package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func preserveEpayConfig(t *testing.T) {
	t.Helper()
	originalGateways := EpayGateways
	originalPayMethods := PayMethods
	originalPayAddress := PayAddress
	originalEpayId := EpayId
	originalEpayKey := EpayKey
	originalCallbackAddress := CustomCallbackAddress
	t.Cleanup(func() {
		EpayGateways = originalGateways
		PayMethods = originalPayMethods
		PayAddress = originalPayAddress
		EpayId = originalEpayId
		EpayKey = originalEpayKey
		CustomCallbackAddress = originalCallbackAddress
	})
}

func TestUpdateEpayGatewaysByJsonStringRejectsUnusableGateways(t *testing.T) {
	validGateway := `{"id":"gw_one","name":"Gateway One","pay_address":"https://pay.example.com","epay_id":"1001","epay_key":"secret"}`

	testCases := []struct {
		name      string
		json      string
		wantError bool
	}{
		{
			name: "accepts a complete gateway",
			json: "[" + validGateway + "]",
		},
		{
			name: "accepts an empty list",
			json: "[]",
		},
		{
			name: "accepts a blank value as an empty list",
			json: "   ",
		},
		{
			name:      "rejects malformed json",
			json:      "{",
			wantError: true,
		},
		{
			name:      "rejects a blank id",
			json:      `[{"id":"","name":"n","pay_address":"https://pay.example.com","epay_id":"1","epay_key":"k"}]`,
			wantError: true,
		},
		{
			name:      "rejects an id that is unsafe in a callback path",
			json:      `[{"id":"../evil","name":"n","pay_address":"https://pay.example.com","epay_id":"1","epay_key":"k"}]`,
			wantError: true,
		},
		{
			name:      "rejects duplicate ids",
			json:      "[" + validGateway + "," + validGateway + "]",
			wantError: true,
		},
		{
			name:      "rejects a missing name",
			json:      `[{"id":"gw_one","name":"","pay_address":"https://pay.example.com","epay_id":"1","epay_key":"k"}]`,
			wantError: true,
		},
		{
			name:      "rejects a missing merchant key",
			json:      `[{"id":"gw_one","name":"n","pay_address":"https://pay.example.com","epay_id":"1","epay_key":""}]`,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			preserveEpayConfig(t)
			EpayGateways = nil

			err := UpdateEpayGatewaysByJsonString(tc.json)
			if tc.wantError {
				require.Error(t, err)
				assert.Empty(t, EpayGateways)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateEpayGatewaysByJsonStringTrimsFields(t *testing.T) {
	preserveEpayConfig(t)

	require.NoError(t, UpdateEpayGatewaysByJsonString(
		`[{"id":" gw_one ","name":" Gateway One ","pay_address":" https://pay.example.com ","epay_id":" 1001 ","epay_key":" secret ","custom_callback_address":" https://cb.example.com "}]`,
	))

	require.Len(t, EpayGateways, 1)
	assert.Equal(t, EpayGateway{
		Id:                    "gw_one",
		Name:                  "Gateway One",
		PayAddress:            "https://pay.example.com",
		EpayId:                "1001",
		EpayKey:               "secret",
		CustomCallbackAddress: "https://cb.example.com",
	}, EpayGateways[0])
}

// The options API withholds merchant keys, so saving the gateway list back must
// keep the stored key instead of wiping it.
func TestMergeEpayGatewaySecretsKeepsStoredKeyWhenBlank(t *testing.T) {
	preserveEpayConfig(t)

	EpayGateways = []EpayGateway{{
		Id:         "gw_one",
		Name:       "Gateway One",
		PayAddress: "https://one.example.com",
		EpayId:     "one_id",
		EpayKey:    "stored_key",
	}}

	merged, err := MergeEpayGatewaySecrets(
		`[{"id":"gw_one","name":"Gateway One Renamed","pay_address":"https://one.example.com","epay_id":"one_id","epay_key":""}]`,
	)
	require.NoError(t, err)
	require.NoError(t, UpdateEpayGatewaysByJsonString(merged))
	require.Len(t, EpayGateways, 1)
	assert.Equal(t, "stored_key", EpayGateways[0].EpayKey)
	assert.Equal(t, "Gateway One Renamed", EpayGateways[0].Name)

	rotated, err := MergeEpayGatewaySecrets(
		`[{"id":"gw_one","name":"Gateway One","pay_address":"https://one.example.com","epay_id":"one_id","epay_key":"rotated_key"}]`,
	)
	require.NoError(t, err)
	require.NoError(t, UpdateEpayGatewaysByJsonString(rotated))
	assert.Equal(t, "rotated_key", EpayGateways[0].EpayKey)

	_, err = MergeEpayGatewaySecrets(
		`[{"id":"gw_new","name":"Gateway New","pay_address":"https://new.example.com","epay_id":"new_id","epay_key":""}]`,
	)
	require.Error(t, err)
}

func TestEpayGatewaysRedactedJSONRemovesMerchantKeys(t *testing.T) {
	preserveEpayConfig(t)

	EpayGateways = []EpayGateway{{
		Id:         "gw_one",
		Name:       "Gateway One",
		PayAddress: "https://one.example.com",
		EpayId:     "one_id",
		EpayKey:    "stored_key",
	}}

	assert.NotContains(t, EpayGatewaysRedactedJSON(), "stored_key")
	assert.Contains(t, EpayGatewaysRedactedJSON(), "gw_one")
	assert.Equal(t, "stored_key", EpayGateways[0].EpayKey)
}

func TestGetEpayGatewayResolvesDefaultAndAdditionalGateways(t *testing.T) {
	preserveEpayConfig(t)

	PayAddress = "https://default.example.com"
	EpayId = "default_id"
	EpayKey = "default_key"
	CustomCallbackAddress = "https://default-callback.example.com"
	EpayGateways = []EpayGateway{{
		Id:         "gw_one",
		Name:       "Gateway One",
		PayAddress: "https://one.example.com",
		EpayId:     "one_id",
		EpayKey:    "one_key",
	}}

	defaultGateway, ok := GetEpayGateway(DefaultEpayGatewayId)
	require.True(t, ok)
	assert.Equal(t, EpayGateway{
		PayAddress:            "https://default.example.com",
		EpayId:                "default_id",
		EpayKey:               "default_key",
		CustomCallbackAddress: "https://default-callback.example.com",
	}, defaultGateway)

	additionalGateway, ok := GetEpayGateway("gw_one")
	require.True(t, ok)
	assert.Equal(t, "https://one.example.com", additionalGateway.PayAddress)

	_, ok = GetEpayGateway("gw_missing")
	assert.False(t, ok)
}

func TestHasConfiguredEpayGatewayCoversDefaultAndAdditionalGateways(t *testing.T) {
	preserveEpayConfig(t)

	PayAddress = ""
	EpayId = ""
	EpayKey = ""
	EpayGateways = nil
	assert.False(t, HasConfiguredEpayGateway())

	EpayGateways = []EpayGateway{{
		Id:         "gw_one",
		Name:       "Gateway One",
		PayAddress: "https://one.example.com",
		EpayId:     "one_id",
		EpayKey:    "one_key",
	}}
	assert.True(t, HasConfiguredEpayGateway())

	EpayGateways = nil
	PayAddress = "https://default.example.com"
	EpayId = "default_id"
	EpayKey = "default_key"
	assert.True(t, HasConfiguredEpayGateway())
}

// The same epay type may be configured once per gateway, so a payment method
// is only identified by the type and gateway pair.
func TestFindPayMethodMatchesTypeAndGateway(t *testing.T) {
	preserveEpayConfig(t)

	PayMethods = []map[string]string{
		{"name": "Alipay", "type": "alipay"},
		{"name": "Alipay Backup", "type": "alipay", "gateway_id": "gw_one"},
		{"name": "Stripe", "type": "stripe"},
	}

	testCases := []struct {
		name      string
		method    string
		gatewayId string
		wantName  string
		wantFound bool
	}{
		{
			name:      "blank gateway matches the default gateway method",
			method:    "alipay",
			gatewayId: "",
			wantName:  "Alipay",
			wantFound: true,
		},
		{
			name:      "gateway id selects the method bound to that gateway",
			method:    "alipay",
			gatewayId: "gw_one",
			wantName:  "Alipay Backup",
			wantFound: true,
		},
		{
			name:      "unconfigured gateway is not accepted",
			method:    "alipay",
			gatewayId: "gw_two",
		},
		{
			name:      "unknown type is not accepted",
			method:    "unionpay",
			gatewayId: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payMethod, found := FindPayMethod(tc.method, tc.gatewayId)
			assert.Equal(t, tc.wantFound, found)
			if tc.wantFound {
				assert.Equal(t, tc.wantName, payMethod["name"])
			}
		})
	}
}
