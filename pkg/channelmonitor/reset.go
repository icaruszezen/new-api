package channelmonitor

import "github.com/QuantumNous/new-api/model"

// ResetMonitorData 清空单个监控项的历史数据，配置本身保留。
// 先清内存采样桶再删库，避免公开状态页仍画出未落库的旧格。
func ResetMonitorData(monitorId string) error {
	if monitorId == "" {
		return nil
	}

	persistMu.Lock()
	defer persistMu.Unlock()

	hotBeats.Range(func(key, _ any) bool {
		if key.(beatKey).monitorId == monitorId {
			hotBeats.Delete(key)
		}
		return true
	})

	if err := model.DeleteChannelMonitorData(monitorId); err != nil {
		return err
	}
	InvalidateStatusCache()
	return nil
}
