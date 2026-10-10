package controller

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/modeliqcheck"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"
)

const (
	// modelIQCheckWorkers 限制同时进行的检查数：每次检查可能持续数分钟并占用上游并发。
	modelIQCheckWorkers = 3
	// modelIQCheckTimeoutGrace 是超时后等待测试链路交回部分输出的时间。
	// 中转只把流读取绑定到 context，上游请求本身不受其约束，因此不能无限等待。
	modelIQCheckTimeoutGrace = 5 * time.Second
)

// modelIQCheckOptionKeys 是执行检查前从数据库刷新的配置项。
var modelIQCheckOptionKeys = []string{
	model_iq_check_setting.EnabledOptionKey,
	model_iq_check_setting.ChecksOptionKey,
	model_iq_check_setting.HistoryLimitOptionKey,
	model_iq_check_setting.RequestTimeoutOptionKey,
}

type modelIQCheckSummary struct {
	Checks    int `json:"checks"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

// runModelIQCheckTask 运行所有到期的智商检查。检查以有界并发执行，
// 进度与汇总只在调用方 goroutine 归并，因此 report 无需并发安全。
func runModelIQCheckTask(ctx context.Context, report func(processed, total int)) (modelIQCheckSummary, error) {
	summary := modelIQCheckSummary{}
	refreshModelIQCheckSettings()
	setting := model_iq_check_setting.GetSetting()
	if !setting.Enabled {
		return summary, nil
	}

	// 清理已被删除的检查项残留数据。只在配置能被解析时执行，避免一次错误写入清空历史。
	if configured, err := modeliqcheck.ParseChecks(setting.Checks); err == nil {
		checkIds := make([]string, 0, len(configured))
		for _, check := range configured {
			checkIds = append(checkIds, check.Id)
		}
		if err := model.DeleteModelIQCheckDataExcept(checkIds); err != nil {
			common.SysError("failed to clean up removed model iq checks: " + err.Error())
		}
	}

	due, err := modeliqcheck.DueChecks(time.Now().Unix())
	if err != nil {
		return summary, err
	}
	summary.Checks = len(due)
	if report != nil {
		report(0, len(due))
	}
	if len(due) == 0 {
		return summary, nil
	}

	userID, err := resolveChannelTestUserID(nil)
	if err != nil {
		return summary, err
	}

	jobs := make(chan modeliqcheck.Check)
	statuses := make(chan int)
	var workers sync.WaitGroup
	for range min(modelIQCheckWorkers, len(due)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for check := range jobs {
				statuses <- runModelIQCheck(ctx, check, userID, setting)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, check := range due {
			select {
			case <-ctx.Done():
				return
			case jobs <- check:
			}
		}
	}()
	go func() {
		workers.Wait()
		close(statuses)
	}()

	processed := 0
	for status := range statuses {
		processed++
		switch status {
		case model.ModelIQCheckStatusSuccess:
			summary.Succeeded++
		case model.ModelIQCheckStatusFailed:
			summary.Failed++
		}
		if report != nil && ctx.Err() == nil {
			report(processed, len(due))
		}
	}
	return summary, nil
}

// refreshModelIQCheckSettings 在执行前读取最新配置。管理员的保存可能落在其它节点，
// 本节点的配置要等下一次 SyncOptions 才更新；用旧配置运行会以旧提示词消耗掉立即运行请求。
func refreshModelIQCheckSettings() {
	if err := model.ReloadOptions(modelIQCheckOptionKeys); err != nil {
		common.SysError("failed to reload model iq check settings: " + err.Error())
	}
}

// runModelIQCheck 执行一次检查并落库，返回结果状态。检查可能在队列中等待多时，
// 开始前按最新配置重新确认：已停用或删除的检查项跳过，修改过的按新配置运行。
// 任务租约丢失（ctx 取消）时不写结果，返回 0；开始时间已写入而结束时间没有，
// 超过运行窗口后调度器会把它视为中断并重跑。
func runModelIQCheck(ctx context.Context, check modeliqcheck.Check, userID int, setting model_iq_check_setting.ModelIQCheckSetting) int {
	if ctx.Err() != nil {
		return 0
	}
	refreshModelIQCheckSettings()
	current, found := modeliqcheck.FindCheck(check.Id)
	if !found || !current.Enabled || !model_iq_check_setting.IsEnabled() {
		return 0
	}
	check = current

	startedAt := time.Now()
	if err := model.MarkModelIQCheckStarted(check.Id, startedAt.Unix()); err != nil {
		common.SysError(fmt.Sprintf("failed to mark model iq check %s started: %s", check.Id, err.Error()))
	}

	result := model.ModelIQCheckResult{
		CheckId:   check.Id,
		ModelName: check.Model,
		Prompt:    check.Prompt,
		Status:    model.ModelIQCheckStatusFailed,
	}
	channel, err := selectChannelForGroupModel(ctx, check.Group, check.Model)
	switch {
	case err != nil:
		result.ErrorType = modeliqcheck.ErrorTypeNoChannel
		result.ErrorMessage = err.Error()
	case shouldSkipChannelMonitorSyntheticProbe(channel, check.Model):
		result.ChannelId = channel.Id
		result.ErrorType = modeliqcheck.ErrorTypeUnsupportedModel
		result.ErrorMessage = fmt.Sprintf("model %s is not a chat model and cannot answer a text prompt", check.Model)
	default:
		result.ChannelId = channel.Id
		executeModelIQCheck(ctx, check, channel, userID, setting.RequestTimeoutSeconds, &result)
	}
	result.DurationMs = time.Since(startedAt).Milliseconds()
	result.ErrorMessage = modeliqcheck.TruncateErrorMessage(result.ErrorMessage)

	if ctx.Err() != nil {
		return 0
	}
	// 运行期间检查项被删除时不再落库：删除时的清理已经执行过，此时写入会留下无人清理的结果与状态。
	refreshModelIQCheckSettings()
	if _, found := modeliqcheck.FindCheck(check.Id); !found {
		return 0
	}

	result.CheckedAt = time.Now().Unix()
	if err := model.SaveModelIQCheckResult(&result, setting.HistoryLimit); err != nil {
		common.SysError(fmt.Sprintf("failed to save model iq check %s result: %s", check.Id, err.Error()))
	}
	if err := model.MarkModelIQCheckFinished(check.Id, time.Now().Unix()); err != nil {
		common.SysError(fmt.Sprintf("failed to mark model iq check %s finished: %s", check.Id, err.Error()))
	}
	modeliqcheck.InvalidateStatusCache()
	common.SysLog(fmt.Sprintf("model iq check %s finished: status=%s duration=%dms",
		check.Id, modeliqcheck.ResultStatusName(result.Status), result.DurationMs))
	return result.Status
}

// executeModelIQCheck 复用渠道测试链路发送检查提示词，并把输出、用量与失败原因写入 result。
// 测试链路在独立 goroutine 中运行：超时后只等待一个短暂的宽限期取回部分输出，
// 否则直接记为超时，遗留的 goroutine 结束后自行退出。
func executeModelIQCheck(ctx context.Context, check modeliqcheck.Check, channel *model.Channel, userID int, timeoutSeconds int, result *model.ModelIQCheckResult) {
	opts := channelTestOptions{
		group:           check.Group,
		skipConsumeLog:  true,
		prompt:          check.Prompt,
		captureResponse: true,
		omitMaxTokens:   check.MaxTokens <= 0,
	}
	if check.MaxTokens > 0 {
		opts.maxTokens = common.GetPointer(uint(check.MaxTokens))
	}

	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	done := make(chan testResult, 1)
	go func() {
		done <- testChannel(checkCtx, channel, userID, check.Model, "", true, opts)
	}()

	var testRes testResult
	timedOut := false
	select {
	case testRes = <-done:
	case <-checkCtx.Done():
		timedOut = true
		select {
		case testRes = <-done:
		case <-time.After(modelIQCheckTimeoutGrace):
		}
	}

	output := modeliqcheck.ExtractOutput(testRes.responseBody)
	result.Content = output.Content
	result.ContentTruncated = output.ContentTruncated
	result.Reasoning = output.Reasoning
	result.ReasoningTruncated = output.ReasoningTruncated
	result.FinishReason = output.FinishReason
	if testRes.usage != nil {
		result.PromptTokens = testRes.usage.PromptTokens
		result.CompletionTokens = testRes.usage.CompletionTokens
		result.ReasoningTokens = testRes.usage.CompletionTokenDetails.ReasoningTokens
	}

	switch {
	case timedOut:
		result.ErrorType = modeliqcheck.ErrorTypeTimeout
		result.ErrorMessage = fmt.Sprintf("timed out after %d seconds", timeoutSeconds)
	case testRes.localErr != nil:
		result.ErrorType = modeliqcheck.ErrorTypeUpstream
		result.ErrorMessage = testRes.localErr.Error()
	case testRes.newAPIError != nil:
		result.ErrorType = modeliqcheck.ErrorTypeUpstream
		result.ErrorMessage = testRes.newAPIError.Error()
	case output.ErrorMessage != "":
		result.ErrorType = modeliqcheck.ErrorTypeUpstream
		result.ErrorMessage = output.ErrorMessage
	case strings.TrimSpace(output.Content) == "":
		finishReason := output.FinishReason
		if finishReason == "" {
			finishReason = "unknown"
		}
		result.ErrorType = modeliqcheck.ErrorTypeEmptyResponse
		result.ErrorMessage = fmt.Sprintf("model returned no content (finish reason: %s)", finishReason)
	default:
		result.Status = model.ModelIQCheckStatusSuccess
	}
}
