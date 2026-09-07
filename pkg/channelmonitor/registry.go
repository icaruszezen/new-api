package channelmonitor

import (
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/channel_monitoring_setting"
)

const (
	// MaxMonitors 同时限制探测开销与公开响应体积。
	MaxMonitors = 50
	// MaxMonitorNameLength 与前端表单校验保持一致。
	MaxMonitorNameLength = 64
	MaxMonitorIconLength = 128
)

// registry 缓存已解析的监控项，并按 group|model 建索引供请求热路径查询。
// 配置存在 option 表中并由 SyncOptions 周期性热更新，因此这里以原始 JSON
// 字符串作为版本标记，只在内容变化时重建。
type registry struct {
	mu       sync.RWMutex
	raw      string
	monitors []Monitor
	byPair   map[string]Monitor
}

var monitorRegistry registry

// Monitors 返回全部监控项（含未启用的），按 sort 再按 name 稳定排序。
func Monitors() []Monitor {
	snapshot := monitorRegistry.load()
	out := make([]Monitor, len(snapshot.monitors))
	copy(out, snapshot.monitors)
	return out
}

// EnabledMonitors 返回启用中的监控项。
func EnabledMonitors() []Monitor {
	snapshot := monitorRegistry.load()
	out := make([]Monitor, 0, len(snapshot.monitors))
	for _, monitor := range snapshot.monitors {
		if monitor.Enabled {
			out = append(out, monitor)
		}
	}
	return out
}

// Lookup 按 (group, model) 查找启用中的监控项，供请求埋点在热路径调用。
func Lookup(group string, modelName string) (Monitor, bool) {
	if group == "" || modelName == "" {
		return Monitor{}, false
	}
	snapshot := monitorRegistry.load()
	monitor, ok := snapshot.byPair[pairKey(group, modelName)]
	return monitor, ok
}

type registrySnapshot struct {
	monitors []Monitor
	byPair   map[string]Monitor
}

func (r *registry) load() registrySnapshot {
	raw := channel_monitoring_setting.RawMonitors()

	r.mu.RLock()
	if r.raw == raw {
		snapshot := registrySnapshot{monitors: r.monitors, byPair: r.byPair}
		r.mu.RUnlock()
		return snapshot
	}
	r.mu.RUnlock()

	monitors, err := ParseMonitors(raw)
	if err != nil {
		common.SysError("failed to parse channel monitoring monitors: " + err.Error())
		monitors = nil
	}
	byPair := make(map[string]Monitor, len(monitors))
	for _, monitor := range monitors {
		if monitor.Enabled {
			byPair[pairKey(monitor.Group, monitor.Model)] = monitor
		}
	}

	r.mu.Lock()
	r.raw = raw
	r.monitors = monitors
	r.byPair = byPair
	snapshot := registrySnapshot{monitors: r.monitors, byPair: r.byPair}
	r.mu.Unlock()
	return snapshot
}

func pairKey(group string, modelName string) string {
	return group + "\x00" + modelName
}

// ParseMonitors 解析监控项 JSON 数组并按展示顺序排序。空输入返回空切片。
func ParseMonitors(raw string) ([]Monitor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return []Monitor{}, nil
	}
	var monitors []Monitor
	if err := common.UnmarshalJsonStr(trimmed, &monitors); err != nil {
		return nil, err
	}
	sortMonitors(monitors)
	return monitors, nil
}

// ValidateMonitors 校验管理员提交的监控项，返回归一化后的结果。
// groupExists / modelInGroup 由调用方注入，便于在没有数据库的场景下测试。
func ValidateMonitors(monitors []Monitor, groupExists func(string) bool, modelInGroup func(string, string) bool) ([]Monitor, error) {
	if len(monitors) > MaxMonitors {
		return nil, fmt.Errorf("at most %d channel monitors are allowed", MaxMonitors)
	}

	seenIds := make(map[string]struct{}, len(monitors))
	seenPairs := make(map[string]struct{}, len(monitors))
	normalized := make([]Monitor, 0, len(monitors))

	for index := range monitors {
		monitor := monitors[index]
		monitor.Id = strings.TrimSpace(monitor.Id)
		monitor.Name = strings.TrimSpace(monitor.Name)
		monitor.Group = strings.TrimSpace(monitor.Group)
		monitor.Model = strings.TrimSpace(monitor.Model)
		monitor.Icon = strings.TrimSpace(monitor.Icon)

		if monitor.Name == "" {
			return nil, fmt.Errorf("monitor #%d: display name is required", index+1)
		}
		if len([]rune(monitor.Name)) > MaxMonitorNameLength {
			return nil, fmt.Errorf("monitor %q: display name must be at most %d characters", monitor.Name, MaxMonitorNameLength)
		}
		if len(monitor.Icon) > MaxMonitorIconLength {
			return nil, fmt.Errorf("monitor %q: icon name is too long", monitor.Name)
		}
		if monitor.Group == "" {
			return nil, fmt.Errorf("monitor %q: group is required", monitor.Name)
		}
		if monitor.Model == "" {
			return nil, fmt.Errorf("monitor %q: model is required", monitor.Name)
		}
		if groupExists != nil && !groupExists(monitor.Group) {
			return nil, fmt.Errorf("monitor %q: group %q does not exist", monitor.Name, monitor.Group)
		}
		if modelInGroup != nil && !modelInGroup(monitor.Group, monitor.Model) {
			return nil, fmt.Errorf("monitor %q: model %q is not available in group %q", monitor.Name, monitor.Model, monitor.Group)
		}

		pair := pairKey(monitor.Group, monitor.Model)
		if _, exists := seenPairs[pair]; exists {
			return nil, fmt.Errorf("duplicated monitor for group %q and model %q", monitor.Group, monitor.Model)
		}
		seenPairs[pair] = struct{}{}

		if monitor.Id == "" {
			monitor.Id = common.GetRandomString(16)
		}
		if len(monitor.Id) > 64 {
			return nil, fmt.Errorf("monitor %q: id is too long", monitor.Name)
		}
		if _, exists := seenIds[monitor.Id]; exists {
			return nil, fmt.Errorf("duplicated monitor id %q", monitor.Id)
		}
		seenIds[monitor.Id] = struct{}{}

		monitor.Sort = index
		normalized = append(normalized, monitor)
	}

	return normalized, nil
}

func sortMonitors(monitors []Monitor) {
	// 简单插入排序即可：监控项数量上限为 50，且需要保持同 sort 值的原始顺序。
	for i := 1; i < len(monitors); i++ {
		for j := i; j > 0 && monitors[j].Sort < monitors[j-1].Sort; j-- {
			monitors[j], monitors[j-1] = monitors[j-1], monitors[j]
		}
	}
}
