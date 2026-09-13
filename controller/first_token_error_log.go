package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func firstTokenErrorQueryFromContext(c *gin.Context) model.FirstTokenErrorLogQuery {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	return model.FirstTokenErrorLogQuery{
		Username:  c.Query("username"),
		TokenName: c.Query("token_name"),
		ModelName: c.Query("model_name"),
		Channel:   channel,
		Group:     c.Query("group"),
		RequestId: c.Query("request_id"),
		StartTs:   startTimestamp,
		EndTs:     endTimestamp,
		StartIdx:  pageInfo.GetStartIdx(),
		PageSize:  pageInfo.GetPageSize(),
	}
}

func GetFirstTokenErrorLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logs, total, err := model.GetFirstTokenErrorLogs(firstTokenErrorQueryFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

func GetFirstTokenErrorLogsStat(c *gin.Context) {
	stat, err := model.GetFirstTokenErrorStat(firstTokenErrorQueryFromContext(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stat)
}

func GetFirstTokenErrorLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}
	log, err := model.GetFirstTokenErrorLogById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "not found"})
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, log)
}

func GetFirstTokenErrorBody(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}
	log, err := model.GetFirstTokenErrorLogById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "not found"})
			return
		}
		common.ApiError(c, err)
		return
	}
	if !log.HasRequestBody {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "request body not available"})
		return
	}
	body, err := model.GetFirstTokenErrorBodyByLogId(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "request body not available"})
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, body)
}
