package modeliqcheck

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/setting/model_iq_check_setting"
)

// registry 缓存已解析的检查项。配置存在 option 表中并由 SyncOptions 周期性热更新，
// 因此这里以原始 JSON 字符串作为版本标记，只在内容变化时重建。
type registry struct {
	mu     sync.RWMutex
	raw    string
	checks []Check
}

var checkRegistry registry

// Checks 返回全部检查项（含未启用的），按 sort 稳定排序。
func Checks() []Check {
	snapshot := checkRegistry.load()
	out := make([]Check, len(snapshot))
	copy(out, snapshot)
	return out
}

// EnabledChecks 返回启用中的检查项。
func EnabledChecks() []Check {
	snapshot := checkRegistry.load()
	out := make([]Check, 0, len(snapshot))
	for _, check := range snapshot {
		if check.Enabled {
			out = append(out, check)
		}
	}
	return out
}

// FindCheck 按 id 查找检查项（含未启用的）。
func FindCheck(id string) (Check, bool) {
	if id == "" {
		return Check{}, false
	}
	for _, check := range checkRegistry.load() {
		if check.Id == id {
			return check, true
		}
	}
	return Check{}, false
}

func (r *registry) load() []Check {
	raw := model_iq_check_setting.RawChecks()

	r.mu.RLock()
	if r.raw == raw {
		checks := r.checks
		r.mu.RUnlock()
		return checks
	}
	r.mu.RUnlock()

	checks, err := ParseChecks(raw)
	if err != nil {
		// 解析失败时保留上一份有效配置，避免一次错误写入清空所有检查。
		common.SysError("failed to parse model iq checks: " + err.Error())
		r.mu.RLock()
		previous := r.checks
		r.mu.RUnlock()
		return previous
	}

	r.mu.Lock()
	r.raw = raw
	r.checks = checks
	r.mu.Unlock()
	return checks
}

// ParseChecks 解析检查项 JSON 数组并按展示顺序排序。空输入返回空切片。
func ParseChecks(raw string) ([]Check, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return []Check{}, nil
	}
	var checks []Check
	if err := common.UnmarshalJsonStr(trimmed, &checks); err != nil {
		return nil, err
	}
	sortChecks(checks)
	return checks, nil
}

// ValidationError 携带 i18n 消息键与模板参数，管理端接口据此返回本地化提示；
// Error() 返回英文文本，供直接写 option 等不做本地化的路径使用。
type ValidationError struct {
	Key     string
	Params  map[string]any
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

func newValidationError(key string, params map[string]any, format string, args ...any) *ValidationError {
	return &ValidationError{
		Key:     key,
		Params:  params,
		message: fmt.Sprintf(format, args...),
	}
}

// ValidateChecks 校验管理员提交的检查项，返回归一化后的结果。
// groupExists / modelInGroup 由调用方注入，便于在没有数据库的场景下测试；传 nil 跳过对应校验。
// 同一分组同一模型允许出现多次，以便用不同提示词考察同一个模型。
func ValidateChecks(checks []Check, groupExists func(string) bool, modelInGroup func(string, string) bool) ([]Check, error) {
	if len(checks) > MaxChecks {
		return nil, newValidationError(i18n.MsgModelIQCheckTooMany, map[string]any{"Max": MaxChecks},
			"at most %d IQ checks are allowed", MaxChecks)
	}

	seenIds := make(map[string]struct{}, len(checks))
	normalized := make([]Check, 0, len(checks))

	for index := range checks {
		check := checks[index]
		check.Id = strings.TrimSpace(check.Id)
		check.Name = strings.TrimSpace(check.Name)
		check.Group = strings.TrimSpace(check.Group)
		check.Model = strings.TrimSpace(check.Model)
		check.Icon = strings.TrimSpace(check.Icon)
		check.Prompt = strings.TrimSpace(check.Prompt)

		if check.Name == "" {
			return nil, newValidationError(i18n.MsgModelIQCheckNameRequired, map[string]any{"Index": index + 1},
				"IQ check #%d: display name is required", index+1)
		}
		if len([]rune(check.Name)) > MaxNameLength {
			return nil, newValidationError(i18n.MsgModelIQCheckNameTooLong, map[string]any{"Name": check.Name, "Max": MaxNameLength},
				"IQ check %q: display name must be at most %d characters", check.Name, MaxNameLength)
		}
		if check.Group == "" {
			return nil, newValidationError(i18n.MsgModelIQCheckGroupRequired, map[string]any{"Name": check.Name},
				"IQ check %q: group is required", check.Name)
		}
		if check.Model == "" {
			return nil, newValidationError(i18n.MsgModelIQCheckModelRequired, map[string]any{"Name": check.Name},
				"IQ check %q: model is required", check.Name)
		}
		if groupExists != nil && !groupExists(check.Group) {
			return nil, newValidationError(i18n.MsgModelIQCheckGroupNotFound, map[string]any{"Name": check.Name, "Group": check.Group},
				"IQ check %q: group %q does not exist", check.Name, check.Group)
		}
		if modelInGroup != nil && !modelInGroup(check.Group, check.Model) {
			return nil, newValidationError(i18n.MsgModelIQCheckModelNotInGroup, map[string]any{"Name": check.Name, "Model": check.Model, "Group": check.Group},
				"IQ check %q: model %q is not available in group %q", check.Name, check.Model, check.Group)
		}
		if check.Prompt == "" {
			return nil, newValidationError(i18n.MsgModelIQCheckPromptRequired, map[string]any{"Name": check.Name},
				"IQ check %q: prompt is required", check.Name)
		}
		if len([]rune(check.Prompt)) > MaxPromptLength {
			return nil, newValidationError(i18n.MsgModelIQCheckPromptTooLong, map[string]any{"Name": check.Name, "Max": MaxPromptLength},
				"IQ check %q: prompt must be at most %d characters", check.Name, MaxPromptLength)
		}
		if check.IntervalMinutes == 0 {
			check.IntervalMinutes = DefaultIntervalMinutes
		}
		if check.IntervalMinutes < MinIntervalMinutes || check.IntervalMinutes > MaxIntervalMinutes {
			return nil, newValidationError(i18n.MsgModelIQCheckIntervalInvalid, map[string]any{"Name": check.Name, "Min": MinIntervalMinutes, "Max": MaxIntervalMinutes},
				"IQ check %q: check interval must be between %d and %d minutes", check.Name, MinIntervalMinutes, MaxIntervalMinutes)
		}
		if check.MaxTokens < 0 || check.MaxTokens > MaxMaxTokens {
			return nil, newValidationError(i18n.MsgModelIQCheckMaxTokensInvalid, map[string]any{"Name": check.Name, "Max": MaxMaxTokens},
				"IQ check %q: max output tokens must be between 0 and %d", check.Name, MaxMaxTokens)
		}
		check.EndpointType = strings.ToLower(strings.TrimSpace(check.EndpointType))
		if !slices.Contains(EndpointTypes, check.EndpointType) {
			return nil, newValidationError(i18n.MsgModelIQCheckEndpointInvalid, map[string]any{"Name": check.Name, "Value": check.EndpointType},
				"IQ check %q: unsupported request format %q", check.Name, check.EndpointType)
		}
		check.ReasoningEffort = strings.ToLower(strings.TrimSpace(check.ReasoningEffort))
		if !slices.Contains(ReasoningEfforts, check.ReasoningEffort) {
			return nil, newValidationError(i18n.MsgModelIQCheckEffortInvalid, map[string]any{"Name": check.Name, "Value": check.ReasoningEffort},
				"IQ check %q: unsupported reasoning effort %q", check.Name, check.ReasoningEffort)
		}
		if len(check.Icon) > MaxIconLength {
			return nil, newValidationError(i18n.MsgModelIQCheckIconTooLong, map[string]any{"Name": check.Name},
				"IQ check %q: icon name is too long", check.Name)
		}

		if check.Id == "" {
			check.Id = common.GetRandomString(16)
		}
		if len(check.Id) > MaxIdLength {
			return nil, newValidationError(i18n.MsgModelIQCheckIdTooLong, map[string]any{"Name": check.Name},
				"IQ check %q: id is too long", check.Name)
		}
		if _, exists := seenIds[check.Id]; exists {
			return nil, newValidationError(i18n.MsgModelIQCheckIdDuplicated, map[string]any{"Id": check.Id},
				"duplicated IQ check id %q", check.Id)
		}
		seenIds[check.Id] = struct{}{}

		check.Sort = index
		normalized = append(normalized, check)
	}

	return normalized, nil
}

// intervalMinutesOf 返回生效的检查间隔，兼容未经校验写入的 0 值。
func intervalMinutesOf(check Check) int {
	if check.IntervalMinutes <= 0 {
		return DefaultIntervalMinutes
	}
	return check.IntervalMinutes
}

func sortChecks(checks []Check) {
	// 简单插入排序即可：检查项数量上限为 50，且需要保持同 sort 值的原始顺序。
	for i := 1; i < len(checks); i++ {
		for j := i; j > 0 && checks[j].Sort < checks[j-1].Sort; j-- {
			checks[j], checks[j-1] = checks[j-1], checks[j]
		}
	}
}
