package operation_setting

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const EpayGatewaysOptionKey = "EpayGateways"

// DefaultEpayGatewayId is the reserved id of the legacy single-gateway
// configuration (PayAddress/EpayId/EpayKey). It stays blank so callback URLs
// and in-flight orders created before multi-gateway support keep working.
const DefaultEpayGatewayId = ""

// maxEpayGateways bounds the configurable gateway list so a malformed bulk
// import cannot blow up option storage or the top-up info payload.
const maxEpayGateways = 50

// epayGatewayIdPattern keeps ids safe to embed in a callback URL path segment.
var epayGatewayIdPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// EpayGateway is one additional 易支付 gateway. Every field mirrors the legacy
// default gateway so each gateway supports the same settings and API calls.
type EpayGateway struct {
	Id                    string `json:"id"`
	Name                  string `json:"name"`
	PayAddress            string `json:"pay_address"`
	EpayId                string `json:"epay_id"`
	EpayKey               string `json:"epay_key"`
	CustomCallbackAddress string `json:"custom_callback_address,omitempty"`
}

var EpayGateways []EpayGateway

// IsConfigured reports whether the gateway can actually reach an upstream.
func (gateway EpayGateway) IsConfigured() bool {
	return strings.TrimSpace(gateway.PayAddress) != "" &&
		strings.TrimSpace(gateway.EpayId) != "" &&
		strings.TrimSpace(gateway.EpayKey) != ""
}

// GetEpayGateway resolves a gateway by id. The blank id maps to the legacy
// default gateway backed by PayAddress/EpayId/EpayKey. The second return value
// reports whether the id refers to a known gateway.
func GetEpayGateway(id string) (EpayGateway, bool) {
	if id == DefaultEpayGatewayId {
		return EpayGateway{
			PayAddress:            PayAddress,
			EpayId:                EpayId,
			EpayKey:               EpayKey,
			CustomCallbackAddress: CustomCallbackAddress,
		}, true
	}
	for _, gateway := range EpayGateways {
		if gateway.Id == id {
			return gateway, true
		}
	}
	return EpayGateway{}, false
}

// HasConfiguredEpayGateway reports whether at least one gateway (default or
// additional) holds complete credentials.
func HasConfiguredEpayGateway() bool {
	if defaultGateway, ok := GetEpayGateway(DefaultEpayGatewayId); ok && defaultGateway.IsConfigured() {
		return true
	}
	for _, gateway := range EpayGateways {
		if gateway.IsConfigured() {
			return true
		}
	}
	return false
}

// parseEpayGateways normalizes the stored JSON and checks everything except the
// credentials, which may still be pending a merge with the stored secrets.
func parseEpayGateways(jsonString string) ([]EpayGateway, error) {
	trimmed := strings.TrimSpace(jsonString)
	if trimmed == "" {
		return []EpayGateway{}, nil
	}
	gateways := make([]EpayGateway, 0)
	if err := common.Unmarshal([]byte(trimmed), &gateways); err != nil {
		return nil, fmt.Errorf("易支付网关配置格式错误: %s", err.Error())
	}
	if len(gateways) > maxEpayGateways {
		return nil, fmt.Errorf("易支付网关数量不能超过 %d 个", maxEpayGateways)
	}
	seenIds := make(map[string]bool, len(gateways))
	for i := range gateways {
		gateway := &gateways[i]
		gateway.Id = strings.TrimSpace(gateway.Id)
		gateway.Name = strings.TrimSpace(gateway.Name)
		gateway.PayAddress = strings.TrimSpace(gateway.PayAddress)
		gateway.EpayId = strings.TrimSpace(gateway.EpayId)
		gateway.EpayKey = strings.TrimSpace(gateway.EpayKey)
		gateway.CustomCallbackAddress = strings.TrimSpace(gateway.CustomCallbackAddress)

		if !epayGatewayIdPattern.MatchString(gateway.Id) {
			return nil, fmt.Errorf("易支付网关 ID %q 非法，仅支持字母、数字、下划线和短横线", gateway.Id)
		}
		if seenIds[gateway.Id] {
			return nil, fmt.Errorf("易支付网关 ID %q 重复", gateway.Id)
		}
		seenIds[gateway.Id] = true
		if gateway.Name == "" {
			return nil, fmt.Errorf("易支付网关 %q 缺少名称", gateway.Id)
		}
	}
	return gateways, nil
}

func decodeEpayGateways(jsonString string) ([]EpayGateway, error) {
	gateways, err := parseEpayGateways(jsonString)
	if err != nil {
		return nil, err
	}
	for _, gateway := range gateways {
		if !gateway.IsConfigured() {
			return nil, fmt.Errorf("易支付网关 %q 缺少支付地址、商户 ID 或商户密钥", gateway.Id)
		}
	}
	return gateways, nil
}

func ValidateEpayGatewaysJSON(jsonString string) error {
	_, err := decodeEpayGateways(jsonString)
	return err
}

func UpdateEpayGatewaysByJsonString(jsonString string) error {
	gateways, err := decodeEpayGateways(jsonString)
	if err != nil {
		return err
	}
	EpayGateways = gateways
	return nil
}

// MergeEpayGatewaySecrets carries the stored merchant key over to gateways whose
// incoming key is blank. The options API withholds the keys, so a blank key from
// the admin UI means "keep the current one" rather than "clear it". It returns
// the JSON to persist.
func MergeEpayGatewaySecrets(jsonString string) (string, error) {
	gateways, err := parseEpayGateways(jsonString)
	if err != nil {
		return "", err
	}
	storedKeys := make(map[string]string, len(EpayGateways))
	for _, gateway := range EpayGateways {
		storedKeys[gateway.Id] = gateway.EpayKey
	}
	for i := range gateways {
		if gateways[i].EpayKey == "" {
			gateways[i].EpayKey = storedKeys[gateways[i].Id]
		}
		if !gateways[i].IsConfigured() {
			return "", fmt.Errorf("易支付网关 %q 缺少支付地址、商户 ID 或商户密钥", gateways[i].Id)
		}
	}
	jsonBytes, err := common.Marshal(gateways)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// EpayGatewaysRedactedJSON returns the gateway list without merchant keys, so
// the options API keeps withholding secrets the way it does for EpayKey.
func EpayGatewaysRedactedJSON() string {
	if len(EpayGateways) == 0 {
		return "[]"
	}
	redacted := make([]EpayGateway, len(EpayGateways))
	copy(redacted, EpayGateways)
	for i := range redacted {
		redacted[i].EpayKey = ""
	}
	jsonBytes, err := common.Marshal(redacted)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func EpayGateways2JsonString() string {
	if len(EpayGateways) == 0 {
		return "[]"
	}
	jsonBytes, err := common.Marshal(EpayGateways)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

// PayMethodGatewayId returns the epay gateway a configured payment method is
// bound to. A missing binding means the legacy default gateway.
func PayMethodGatewayId(payMethod map[string]string) string {
	return strings.TrimSpace(payMethod["gateway_id"])
}

// FindPayMethod looks up a configured payment method by its type and the epay
// gateway it is bound to. The same type may be configured once per gateway, so
// both parts are required to identify a method.
func FindPayMethod(method string, gatewayId string) (map[string]string, bool) {
	for _, payMethod := range PayMethods {
		if payMethod["type"] == method && PayMethodGatewayId(payMethod) == strings.TrimSpace(gatewayId) {
			return payMethod, true
		}
	}
	return nil, false
}
