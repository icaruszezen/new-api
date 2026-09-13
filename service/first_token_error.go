package service

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/gin-gonic/gin"
)

func CalculateWouldBeTextQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) int {
	if relayInfo == nil {
		return 0
	}
	billingUsage := effectiveBillingUsage(usage)
	summary := calculateTextQuotaSummary(ctx, relayInfo, billingUsage)
	if usage != nil {
		var tieredUsedVars map[string]bool
		if snap := relayInfo.TieredBillingSnapshot; snap != nil {
			tieredUsedVars = billingexpr.UsedVars(snap.ExprString)
		}
		if tieredOk, tieredQuota, tieredRes := TryTieredSettle(relayInfo, BuildTieredTokenParams(billingUsage, summary.IsClaudeUsageSemantic, tieredUsedVars)); tieredOk {
			return composeTieredTextQuota(relayInfo, summary, tieredQuota, tieredRes)
		}
	}
	return summary.Quota
}

func HandleResponsesFirstTokenError(c *gin.Context, info *relaycommon.RelayInfo, dirty *dto.Usage, errorMessage string) {
	if info == nil {
		return
	}
	info.FirstTokenErrorHandled = true
	originalEndReason := relaycommon.StreamEndReasonNone
	if info.StreamStatus != nil {
		originalEndReason = info.StreamStatus.EndReason
		info.StreamStatus.OverrideEndReason(relaycommon.StreamEndReasonFirstTokenError, nil)
	}

	useTimeSeconds := 0
	if !info.StartTime.IsZero() {
		useTimeSeconds = int(time.Now().Unix() - info.StartTime.Unix())
	}
	modelName := info.OriginModelName
	tokenName := ""
	group := info.UsingGroup
	if c != nil {
		tokenName = c.GetString("token_name")
		if group == "" {
			group = c.GetString("group")
		}
	}
	if common.FirstTokenErrorLogEnabled {
		if err := persistFirstTokenErrorSnapshot(c, info, dirty, errorMessage, originalEndReason, useTimeSeconds, tokenName, group); err != nil {
			model.LogFirstTokenErrorPersistFailure(c, err)
		}
	}
	if c != nil && model.LOG_DB != nil {
		model.RecordFirstTokenUserErrorLog(c, info.UserId, info.GetChannelID(), modelName, tokenName, errorMessage, info.TokenId, useTimeSeconds, info.IsStream, group)
	}
}

func persistFirstTokenErrorSnapshot(c *gin.Context, info *relaycommon.RelayInfo, dirty *dto.Usage, errorMessage string, originalEndReason relaycommon.StreamEndReason, useTimeSeconds int, tokenName string, group string) error {
	wouldBeQuota := CalculateWouldBeTextQuota(c, info, dirty)
	promptTokens := 0
	completionTokens := 0
	cacheTokens := 0
	if dirty != nil {
		promptTokens = dirty.PromptTokens
		completionTokens = dirty.CompletionTokens
		cacheTokens = dirty.PromptTokensDetails.CachedTokens
	}
	frtMs := int64(0)
	if info.FirstResponseTime.After(info.StartTime) {
		frtMs = info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	}
	requestId := info.RequestId
	upstreamRequestId := ""
	username := ""
	channelName := ""
	requestPath := ""
	if c != nil {
		if requestId == "" {
			requestId = c.GetString(common.RequestIdKey)
		}
		upstreamRequestId = c.GetString(common.UpstreamRequestIdKey)
		username = c.GetString("username")
		channelName = c.GetString("channel_name")
		if c.Request != nil && c.Request.URL != nil {
			requestPath = c.Request.URL.Path
		}
	}
	dirtyJSON := ""
	if dirty != nil {
		dirtyJSON = common.GetJsonString(dirty)
	}
	streamInfo := map[string]interface{}{
		"status":     "error",
		"end_reason": string(relaycommon.StreamEndReasonFirstTokenError),
	}
	if originalEndReason != "" {
		streamInfo["original_end_reason"] = string(originalEndReason)
	}
	channelId := info.GetChannelID()
	other := map[string]interface{}{
		"channel_id":   channelId,
		"channel_name": channelName,
	}
	if c != nil {
		other["billing_path"] = usageBillingPathForLog(common.GetContextKeyBool(c, constant.ContextKeyLocalCountTokens), dirty)
	}
	log := &model.FirstTokenErrorLog{
		RequestId:         requestId,
		UpstreamRequestId: upstreamRequestId,
		UserId:            info.UserId,
		Username:          username,
		TokenId:           info.TokenId,
		TokenName:         tokenName,
		ChannelId:         channelId,
		ChannelName:       channelName,
		Group:             group,
		ModelName:         info.OriginModelName,
		PromptTokens:      promptTokens,
		CompletionTokens:  completionTokens,
		CacheTokens:       cacheTokens,
		WouldBeQuota:      wouldBeQuota,
		UseTime:           useTimeSeconds,
		IsStream:          info.IsStream,
		FrtMs:             frtMs,
		ErrorMessage:      errorMessage,
		DirtyUsageJson:    dirtyJSON,
		StreamStatusJson:  common.MapToJsonStr(streamInfo),
		RequestPath:       requestPath,
		OtherJson:         common.MapToJsonStr(other),
	}
	if err := model.InsertFirstTokenErrorLog(log); err != nil {
		return err
	}
	if !common.FirstTokenErrorBodyCaptureEnabled || c == nil || log.Id == 0 {
		return nil
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		logger.LogError(c, "first-token error body capture skipped: "+err.Error())
		return nil
	}
	raw, err := storage.Bytes()
	if err != nil {
		logger.LogError(c, "first-token error body capture skipped: "+err.Error())
		return nil
	}
	body, truncated, byteSize := model.PrepareFirstTokenErrorBody(raw)
	if byteSize == 0 {
		return nil
	}
	return model.SaveFirstTokenErrorBody(log.Id, body, truncated, byteSize)
}
