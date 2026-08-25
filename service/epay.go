package service

import (
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}

// GetEpayCallbackAddress returns the base address epay callbacks for the given
// gateway must hit: the gateway's own override first, then the global custom
// callback address, then the server address. The default gateway carries the
// global override, so it resolves exactly like GetCallbackAddress.
func GetEpayCallbackAddress(gateway operation_setting.EpayGateway) string {
	if gateway.CustomCallbackAddress != "" {
		return gateway.CustomCallbackAddress
	}
	return GetCallbackAddress()
}
