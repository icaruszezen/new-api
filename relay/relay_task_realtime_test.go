package relay

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// A client fetch of a Gemini/Vertex task queries the upstream directly. When
// that query is the one that sees the task fail, the task must still be
// refunded: the poller skips finished tasks, so nothing else would.
func TestRealtimeFetchRefundsTaskItFindsFailed(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.Channel{}, &model.Task{}, &model.User{}, &model.Log{}))
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousFactory := service.GetTaskAdaptorFunc
	previousRedis, previousMemoryCache, previousBatch := common.RedisEnabled, common.MemoryCacheEnabled, common.BatchUpdateEnabled
	model.DB, model.LOG_DB = database, database
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled, common.MemoryCacheEnabled, common.BatchUpdateEnabled = false, false, false
	service.GetTaskAdaptorFunc = func(platform constant.TaskPlatform) service.TaskPollingAdaptor {
		if adaptor := GetTaskAdaptor(platform); adaptor != nil {
			return adaptor
		}
		return nil
	}
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		service.GetTaskAdaptorFunc = previousFactory
		common.RedisEnabled, common.MemoryCacheEnabled, common.BatchUpdateEnabled = previousRedis, previousMemoryCache, previousBatch
		if sqlDB, err := database.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"operations/task-1","done":true,"error":{"code":3,"message":"blocked by safety"}}`))
	}))
	t.Cleanup(server.Close)
	channel := &model.Channel{Name: "realtime-refund", Type: constant.ChannelTypeGemini, Key: "channel-key", BaseURL: &server.URL, Status: common.ChannelStatusEnabled}
	require.NoError(t, database.Create(channel).Error)
	require.NoError(t, database.Create(&model.User{Id: 7, Username: "realtime-owner", Quota: 100}).Error)
	task := &model.Task{
		TaskID: "public-task", Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeGemini)),
		UserId: 7, ChannelId: channel.Id, Quota: 1000,
		Status: model.TaskStatusInProgress, Progress: "50%",
		PrivateData: model.TaskPrivateData{
			Key: "channel-key", UpstreamTaskID: base64.RawURLEncoding.EncodeToString([]byte("operations/task-1")),
		},
	}
	require.NoError(t, database.Create(task).Error)

	tryRealtimeFetch(task, true)

	var stored model.Task
	require.NoError(t, database.First(&stored, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), stored.Status)
	assert.Equal(t, "blocked by safety", stored.FailReason)
	assert.Zero(t, stored.Quota, "the failed task is refunded")
	var user model.User
	require.NoError(t, database.First(&user, 7).Error)
	assert.Equal(t, 1100, user.Quota)
}
