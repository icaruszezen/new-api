package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2"`
	UseGroup  string `json:"use_group" gorm:"index;size:64;default:''"`
	TokenID   int    `json:"token_id" gorm:"index;default:0"`
	ChannelID int    `json:"channel_id" gorm:"index;default:0"`
	NodeName  string `json:"node_name" gorm:"index;size:64;default:''"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	// PromptTokens 与 CacheTokens 是缓存读取率的分母与分子，语义与用量日志的
	// prompt_tokens / other.cache_tokens 一致。历史行没有这两列，读取率会偏低。
	PromptTokens int `json:"prompt_tokens" gorm:"default:0"`
	CacheTokens  int `json:"cache_tokens" gorm:"default:0"`
	Count        int `json:"count" gorm:"default:0"`
	Quota        int `json:"quota" gorm:"default:0"`
}

// QuotaDataSummary 用户全时段用量合计，不受看板接口的 30 天窗口限制。
type QuotaDataSummary struct {
	TokenUsed    int64 `json:"token_used"`
	PromptTokens int64 `json:"prompt_tokens"`
	CacheTokens  int64 `json:"cache_tokens"`
	// CacheSampledCount 只数带 prompt_tokens 的桶里的请求，是缓存读取率背后的样本量。
	// 新列上线前的历史桶没有输入 token，把它们算进样本会让老账号立刻用极少的新数据出数。
	CacheSampledCount int64 `json:"cache_sampled_count"`
}

type QuotaDataLogParams struct {
	UserID       int
	Username     string
	ModelName    string
	Quota        int
	CreatedAt    int64
	TokenUsed    int
	PromptTokens int
	CacheTokens  int
	UseGroup     string
	TokenID      int
	ChannelID    int
	NodeName     string
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(quotaData *QuotaData) {
	key := fmt.Sprintf("%d\x00%s\x00%s\x00%d\x00%s\x00%d\x00%d\x00%s",
		quotaData.UserID,
		quotaData.Username,
		quotaData.ModelName,
		quotaData.CreatedAt,
		quotaData.UseGroup,
		quotaData.TokenID,
		quotaData.ChannelID,
		quotaData.NodeName,
	)
	count := quotaData.Count
	quota := quotaData.Quota
	tokenUsed := quotaData.TokenUsed
	promptTokens := quotaData.PromptTokens
	cacheTokens := quotaData.CacheTokens
	cachedQuotaData, ok := CacheQuotaData[key]
	if ok {
		cachedQuotaData.Count += count
		cachedQuotaData.Quota += quota
		cachedQuotaData.TokenUsed += tokenUsed
		cachedQuotaData.PromptTokens += promptTokens
		cachedQuotaData.CacheTokens += cacheTokens
		quotaData = cachedQuotaData
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(params QuotaDataLogParams) {
	// 只精确到小时
	createdAt := params.CreatedAt - (params.CreatedAt % 3600)
	quotaData := &QuotaData{
		UserID:       params.UserID,
		Username:     params.Username,
		ModelName:    params.ModelName,
		CreatedAt:    createdAt,
		UseGroup:     params.UseGroup,
		TokenID:      params.TokenID,
		ChannelID:    params.ChannelID,
		NodeName:     params.NodeName,
		Count:        1,
		Quota:        params.Quota,
		TokenUsed:    params.TokenUsed,
		PromptTokens: params.PromptTokens,
		CacheTokens:  params.CacheTokens,
	}

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(quotaData)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").
			Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
				quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
			First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(quotaData *QuotaData) {
	err := DB.Table("quota_data").
		Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and channel_id = ? and node_name = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.ChannelID, quotaData.NodeName).
		Updates(map[string]interface{}{
			"count":         gorm.Expr("count + ?", quotaData.Count),
			"quota":         gorm.Expr("quota + ?", quotaData.Quota),
			"token_used":    gorm.Expr("token_used + ?", quotaData.TokenUsed),
			"prompt_tokens": gorm.Expr("prompt_tokens + ?", quotaData.PromptTokens),
			"cache_tokens":  gorm.Expr("cache_tokens + ?", quotaData.CacheTokens),
		}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").
		Select("user_id, username, model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
		Group("user_id, username, model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").
		Select("user_id, username, model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, sum(prompt_tokens) as prompt_tokens, sum(cache_tokens) as cache_tokens").
		Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
		Group("user_id, username, model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

// GetQuotaDataSummaryByUserId 汇总用户全部历史用量，供控制台首页的累计指标使用。
// 合计用 int64，避免长期累积超出单行 int 列的范围。
func GetQuotaDataSummaryByUserId(userId int) (summary QuotaDataSummary, err error) {
	err = DB.Table("quota_data").
		Select("COALESCE(sum(token_used), 0) as token_used, COALESCE(sum(prompt_tokens), 0) as prompt_tokens, COALESCE(sum(cache_tokens), 0) as cache_tokens, COALESCE(sum(CASE WHEN prompt_tokens > 0 THEN count ELSE 0 END), 0) as cache_sampled_count").
		Where("user_id = ?", userId).
		Scan(&summary).Error
	return summary, err
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("username, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime)
	}
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	// only select model_name, sum(count) as count, sum(quota) as quota, model_name, created_at from quota_data group by model_name, created_at;
	//err = DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&quotaDatas).Error
	err = DB.Table("quota_data").Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at").Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&quotaDatas).Error
	return quotaDatas, err
}
