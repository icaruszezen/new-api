package controller

import (
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/gin-gonic/gin"
)

// epayGatewayParam is the route parameter carrying the gateway id on the
// per-gateway callback endpoints.
const epayGatewayParam = "gatewayId"

// resolveEpayGateway returns the configured gateway a request is bound to.
// A blank id means the legacy default gateway (PayAddress/EpayId/EpayKey), so
// payment methods and orders created before multi-gateway support keep working.
func resolveEpayGateway(gatewayId string) (operation_setting.EpayGateway, bool) {
	gateway, ok := operation_setting.GetEpayGateway(strings.TrimSpace(gatewayId))
	if !ok || !gateway.IsConfigured() {
		return operation_setting.EpayGateway{}, false
	}
	return gateway, true
}

// GetEpayClient builds a signing/verifying client for one epay gateway. Every
// gateway keeps its own merchant id and key, so a callback must be verified
// with the very gateway the order was created against.
func GetEpayClient(gateway operation_setting.EpayGateway) *epay.Client {
	if !gateway.IsConfigured() {
		return nil
	}
	client, err := epay.NewClient(&epay.Config{
		PartnerID: gateway.EpayId,
		Key:       gateway.EpayKey,
	}, gateway.PayAddress)
	if err != nil {
		return nil
	}
	return client
}

// epayCallbackURL builds an absolute callback endpoint for the gateway. The
// default gateway keeps the historical suffix-free path so callbacks for
// orders placed before this feature still resolve.
func epayCallbackURL(gateway operation_setting.EpayGateway, path string) (*url.URL, error) {
	if gateway.Id != operation_setting.DefaultEpayGatewayId {
		path = path + "/" + gateway.Id
	}
	return url.Parse(service.GetEpayCallbackAddress(gateway) + path)
}

// epayCallbackGateway resolves the gateway addressed by a callback request. It
// is the only place a callback may pick its verification key from, so an
// unknown gateway id is rejected instead of silently falling back.
func epayCallbackGateway(c *gin.Context) (operation_setting.EpayGateway, bool) {
	return resolveEpayGateway(c.Param(epayGatewayParam))
}

// nonEpayPaymentMethodTypes are payment types served by their own gateway
// instead of 易支付, so they carry no epay gateway binding.
var nonEpayPaymentMethodTypes = map[string]bool{
	model.PaymentMethodStripe:       true,
	model.PaymentMethodCreem:        true,
	model.PaymentMethodWaffo:        true,
	model.PaymentMethodWaffoPancake: true,
}

// payMethodsWithReachableGateway drops epay methods bound to a gateway that is
// missing or incompletely configured, so users never see a button that cannot
// start a payment.
func payMethodsWithReachableGateway(payMethods []map[string]string) []map[string]string {
	reachable := make([]map[string]string, 0, len(payMethods))
	for _, payMethod := range payMethods {
		if nonEpayPaymentMethodTypes[payMethod["type"]] {
			reachable = append(reachable, payMethod)
			continue
		}
		if _, ok := resolveEpayGateway(operation_setting.PayMethodGatewayId(payMethod)); ok {
			reachable = append(reachable, payMethod)
		}
	}
	return reachable
}
