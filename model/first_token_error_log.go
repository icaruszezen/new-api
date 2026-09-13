package model

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FirstTokenErrorLog struct {
	Id                int    `json:"id"`
	CreatedAt         int64  `json:"created_at" gorm:"index"`
	RequestId         string `json:"request_id" gorm:"type:varchar(64);index"`
	UpstreamRequestId string `json:"upstream_request_id" gorm:"type:varchar(128);index"`
	UserId            int    `json:"user_id" gorm:"index"`
	Username          string `json:"username" gorm:"index"`
	TokenId           int    `json:"token_id" gorm:"index"`
	TokenName         string `json:"token_name" gorm:"index"`
	ChannelId         int    `json:"channel_id" gorm:"index"`
	ChannelName       string `json:"channel_name"`
	Group             string `json:"group" gorm:"index"`
	ModelName         string `json:"model_name" gorm:"index"`
	PromptTokens      int    `json:"prompt_tokens"`
	CompletionTokens  int    `json:"completion_tokens"`
	CacheTokens       int    `json:"cache_tokens"`
	WouldBeQuota      int    `json:"would_be_quota"`
	UseTime           int    `json:"use_time"`
	IsStream          bool   `json:"is_stream"`
	FrtMs             int64  `json:"frt_ms"`
	ErrorMessage      string `json:"error_message" gorm:"type:text"`
	DirtyUsageJson    string `json:"dirty_usage_json" gorm:"type:text"`
	StreamStatusJson  string `json:"stream_status_json" gorm:"type:text"`
	RequestPath       string `json:"request_path"`
	OtherJson         string `json:"other_json" gorm:"type:text"`
	HasRequestBody    bool   `json:"has_request_body"`
}

func (FirstTokenErrorLog) TableName() string {
	return "first_token_error_logs"
}

type FirstTokenErrorBody struct {
	Id        int    `json:"id"`
	LogId     int    `json:"log_id" gorm:"index;not null"`
	CreatedAt int64  `json:"created_at" gorm:"index"`
	Body      string `json:"body" gorm:"type:text"`
	Truncated bool   `json:"truncated"`
	ByteSize  int    `json:"byte_size"`
}

func (FirstTokenErrorBody) TableName() string {
	return "first_token_error_bodies"
}

type FirstTokenErrorLogQuery struct {
	Username  string
	TokenName string
	ModelName string
	Channel   int
	Group     string
	RequestId string
	StartTs   int64
	EndTs     int64
	StartIdx  int
	PageSize  int
}

type FirstTokenErrorStat struct {
	Count                    int64 `json:"count"`
	WouldBeQuota             int64 `json:"would_be_quota"`
	BodyCount                int64 `json:"body_count"`
	BodyCaptureEnabled       bool  `json:"body_capture_enabled"`
	CorrectionEnabled        bool  `json:"correction_enabled"`
	TreatAllOutputOneEnabled bool  `json:"treat_all_output_one_enabled"`
	LogEnabled               bool  `json:"log_enabled"`
	LogMaxKeep               int   `json:"log_max_keep"`
}

func validateFirstTokenErrorLogMaxKeep(value string) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("FirstTokenErrorLogMaxKeep must be an integer")
	}
	if n < common.FirstTokenErrorLogMaxKeepMin || n > common.FirstTokenErrorLogMaxKeepMax {
		return fmt.Errorf("FirstTokenErrorLogMaxKeep must be between %d and %d", common.FirstTokenErrorLogMaxKeepMin, common.FirstTokenErrorLogMaxKeepMax)
	}
	return nil
}

func clampFirstTokenErrorLogMaxKeep(n int) int {
	if n < common.FirstTokenErrorLogMaxKeepMin {
		return common.FirstTokenErrorLogMaxKeepDefault
	}
	if n > common.FirstTokenErrorLogMaxKeepMax {
		return common.FirstTokenErrorLogMaxKeepMax
	}
	return n
}

func InsertFirstTokenErrorLog(log *FirstTokenErrorLog) error {
	if DB == nil || log == nil {
		return nil
	}
	if log.CreatedAt == 0 {
		log.CreatedAt = common.GetTimestamp()
	}
	if err := DB.Create(log).Error; err != nil {
		return err
	}
	scheduleFirstTokenErrorLogPrune()
	return nil
}

var firstTokenErrorLogPruneScheduled atomic.Bool

func scheduleFirstTokenErrorLogPrune() {
	if !firstTokenErrorLogPruneScheduled.CompareAndSwap(false, true) {
		return
	}
	gopool.Go(func() {
		defer firstTokenErrorLogPruneScheduled.Store(false)
		if err := PruneFirstTokenErrorLogs(); err != nil {
			common.SysLog("failed to prune first-token error logs: " + err.Error())
		}
	})
}

func SaveFirstTokenErrorBody(logId int, body string, truncated bool, byteSize int) error {
	if DB == nil || logId == 0 {
		return nil
	}
	record := &FirstTokenErrorBody{
		LogId:     logId,
		CreatedAt: common.GetTimestamp(),
		Body:      body,
		Truncated: truncated,
		ByteSize:  byteSize,
	}
	if err := DB.Create(record).Error; err != nil {
		return err
	}
	if err := DB.Model(&FirstTokenErrorLog{}).Where("id = ?", logId).Update("has_request_body", true).Error; err != nil {
		return err
	}
	return PruneFirstTokenErrorBodies()
}

func PruneFirstTokenErrorLogs() error {
	if DB == nil {
		return nil
	}
	maxKeep := common.FirstTokenErrorLogMaxKeep
	if maxKeep < common.FirstTokenErrorLogMaxKeepMin {
		maxKeep = common.FirstTokenErrorLogMaxKeepDefault
	}
	var total int64
	if err := DB.Model(&FirstTokenErrorLog{}).Count(&total).Error; err != nil {
		return err
	}
	if total <= int64(maxKeep) {
		return nil
	}
	extra := int(total) - maxKeep
	var stale []FirstTokenErrorLog
	if err := DB.Model(&FirstTokenErrorLog{}).
		Select("id").
		Order("created_at asc, id asc").
		Limit(extra).
		Find(&stale).Error; err != nil {
		return err
	}
	if len(stale) == 0 {
		return nil
	}
	ids := make([]int, 0, len(stale))
	for i := range stale {
		ids = append(ids, stale[i].Id)
	}
	if err := DB.Where("log_id IN ?", ids).Delete(&FirstTokenErrorBody{}).Error; err != nil {
		return err
	}
	return DB.Where("id IN ?", ids).Delete(&FirstTokenErrorLog{}).Error
}

func PruneFirstTokenErrorBodies() error {
	if DB == nil {
		return nil
	}
	var keepIDs []int
	if err := DB.Model(&FirstTokenErrorBody{}).
		Order("created_at desc, id desc").
		Limit(common.FirstTokenErrorBodyKeep).
		Pluck("id", &keepIDs).Error; err != nil {
		return err
	}
	if len(keepIDs) == 0 {
		return nil
	}
	var stale []FirstTokenErrorBody
	if err := DB.Model(&FirstTokenErrorBody{}).
		Select("id", "log_id").
		Where("id NOT IN ?", keepIDs).
		Find(&stale).Error; err != nil {
		return err
	}
	if len(stale) == 0 {
		return nil
	}
	bodyIDs := make([]int, 0, len(stale))
	logIDs := make([]int, 0, len(stale))
	for i := range stale {
		bodyIDs = append(bodyIDs, stale[i].Id)
		logIDs = append(logIDs, stale[i].LogId)
	}
	if err := DB.Where("id IN ?", bodyIDs).Delete(&FirstTokenErrorBody{}).Error; err != nil {
		return err
	}
	return DB.Model(&FirstTokenErrorLog{}).Where("id IN ?", logIDs).Update("has_request_body", false).Error
}

func firstTokenErrorLogQuery(q FirstTokenErrorLogQuery) (*gorm.DB, error) {
	tx := DB.Model(&FirstTokenErrorLog{})
	var err error
	if tx, err = applyMainDatabaseTextFilter(tx, "username", q.Username); err != nil {
		return nil, err
	}
	if tx, err = applyMainDatabaseTextFilter(tx, "model_name", q.ModelName); err != nil {
		return nil, err
	}
	if q.TokenName != "" {
		tx = tx.Where("token_name = ?", q.TokenName)
	}
	if q.RequestId != "" {
		tx = tx.Where("request_id = ?", q.RequestId)
	}
	if q.StartTs != 0 {
		tx = tx.Where("created_at >= ?", q.StartTs)
	}
	if q.EndTs != 0 {
		tx = tx.Where("created_at <= ?", q.EndTs)
	}
	if q.Channel != 0 {
		tx = tx.Where("channel_id = ?", q.Channel)
	}
	if q.Group != "" {
		tx = tx.Where(commonGroupCol+" = ?", q.Group)
	}
	return tx, nil
}

func GetFirstTokenErrorLogs(q FirstTokenErrorLogQuery) (logs []*FirstTokenErrorLog, total int64, err error) {
	if DB == nil {
		return nil, 0, nil
	}
	tx, err := firstTokenErrorLogQuery(q)
	if err != nil {
		return nil, 0, err
	}
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = common.ItemsPerPage
	}
	err = tx.Order("created_at desc, id desc").Limit(pageSize).Offset(q.StartIdx).Find(&logs).Error
	return logs, total, err
}

func GetFirstTokenErrorLogById(id int) (*FirstTokenErrorLog, error) {
	if DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var log FirstTokenErrorLog
	err := DB.First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func GetFirstTokenErrorBodyByLogId(logId int) (*FirstTokenErrorBody, error) {
	if DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var body FirstTokenErrorBody
	err := DB.Where("log_id = ?", logId).Order("id desc").First(&body).Error
	if err != nil {
		return nil, err
	}
	return &body, nil
}

func GetFirstTokenErrorStat(q FirstTokenErrorLogQuery) (*FirstTokenErrorStat, error) {
	stat := &FirstTokenErrorStat{
		BodyCaptureEnabled:       common.FirstTokenErrorBodyCaptureEnabled,
		CorrectionEnabled:        common.FirstTokenErrorCorrectionEnabled,
		TreatAllOutputOneEnabled: common.FirstTokenErrorTreatAllOutputOneEnabled,
		LogEnabled:               common.FirstTokenErrorLogEnabled,
		LogMaxKeep:               common.FirstTokenErrorLogMaxKeep,
	}
	if DB == nil {
		return stat, nil
	}
	tx, err := firstTokenErrorLogQuery(q)
	if err != nil {
		return nil, err
	}
	if err = tx.Count(&stat.Count).Error; err != nil {
		return nil, err
	}
	sumTx, err := firstTokenErrorLogQuery(q)
	if err != nil {
		return nil, err
	}
	var quota *int64
	if err = sumTx.Select("SUM(would_be_quota)").Scan(&quota).Error; err != nil {
		return nil, err
	}
	if quota != nil {
		stat.WouldBeQuota = *quota
	}
	if err = DB.Model(&FirstTokenErrorBody{}).Count(&stat.BodyCount).Error; err != nil {
		return nil, err
	}
	return stat, nil
}

func CountFirstTokenErrorBodies() int64 {
	if DB == nil {
		return 0
	}
	var n int64
	_ = DB.Model(&FirstTokenErrorBody{}).Count(&n).Error
	return n
}

func PrepareFirstTokenErrorBody(raw []byte) (body string, truncated bool, byteSize int) {
	if len(raw) == 0 {
		return "", false, 0
	}
	byteSize = len(raw)
	if byteSize <= common.FirstTokenErrorBodyMaxBytes {
		return string(raw), false, byteSize
	}
	var payload map[string]any
	if err := common.Unmarshal(raw, &payload); err == nil {
		kept := struct {
			Model        any `json:"model,omitempty"`
			Instructions any `json:"instructions,omitempty"`
			Input        any `json:"input,omitempty"`
			Messages     any `json:"messages,omitempty"`
		}{
			Model:        payload["model"],
			Instructions: payload["instructions"],
			Input:        payload["input"],
			Messages:     payload["messages"],
		}
		encoded, encErr := common.Marshal(kept)
		if encErr == nil && len(encoded) > 0 {
			if len(encoded) <= common.FirstTokenErrorBodyMaxBytes {
				return string(encoded), true, len(encoded)
			}
			truncatedBody := truncateUTF8Bytes(encoded, common.FirstTokenErrorBodyMaxBytes)
			return truncatedBody, true, len(truncatedBody)
		}
	}
	truncatedBody := truncateUTF8Bytes(raw, common.FirstTokenErrorBodyMaxBytes)
	return truncatedBody, true, len(truncatedBody)
}

func truncateUTF8Bytes(raw []byte, maxBytes int) string {
	if len(raw) <= maxBytes {
		return string(raw)
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(raw[cut]) {
		cut--
	}
	return string(raw[:cut])
}

func RecordFirstTokenUserErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int, isStream bool, group string) {
	if c == nil {
		return
	}
	other := map[string]interface{}{}
	if c.Request != nil && c.Request.URL != nil {
		other["request_path"] = c.Request.URL.Path
	}
	RecordErrorLog(c, userId, channelId, modelName, tokenName, content, tokenId, useTimeSeconds, isStream, group, other)
}

func LogFirstTokenErrorPersistFailure(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if c != nil {
		logger.LogError(c, "failed to persist first-token error log: "+err.Error())
		return
	}
	common.SysLog("failed to persist first-token error log: " + err.Error())
}
