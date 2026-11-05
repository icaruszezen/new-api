package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An affinity hit spends attempt 0 on the bound channel without visiting a
// priority tier, so the first fallback must start at the highest priority.
func TestChannelAffinityRetryPriorities(t *testing.T) {
	db := setupChannelSelectAutoGroupsTest(t)
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","empty":"Empty"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"empty":1}`))

	affinity := operation_setting.GetChannelAffinitySetting()
	previousAffinity := *affinity
	t.Cleanup(func() { *affinity = previousAffinity })

	const modelName = "affinity-retry-test"
	const requestPath = "/v1/chat/completions"
	for _, spec := range []struct {
		id       int
		priority int64
		group    string
	}{{24, 4, "default"}, {27, 3, "default"}, {30, 2, "default"}, {34, 4, "vip"}, {37, 3, "vip"}} {
		channel := model.Channel{
			Id: spec.id, Name: fmt.Sprintf("channel-%d", spec.id), Type: constant.ChannelTypeOpenAI,
			Key: "test-only", Status: common.ChannelStatusEnabled, Models: modelName, Group: spec.group,
			Priority: common.GetPointer(spec.priority), Weight: common.GetPointer(uint(1)),
		}
		require.NoError(t, db.Create(&channel).Error)
		require.NoError(t, channel.AddAbilities(db))
	}

	// The priority-tier index is computed in this package and handed to the
	// model layer the same way for the memory cache and the database, so the
	// memory cache is enough here (the database path needs model.InitDB).
	gin.SetMode(gin.TestMode)
	model.InitChannelCache()
	for _, tc := range []struct {
		name       string
		enabled    bool
		skipRetry  bool
		binding    int
		retries    int
		autoGroups []string
		crossGroup bool
		want       []int
	}{
		{name: "ordinary routing", retries: 2, want: []int{24, 27, 30}},
		{name: "affinity miss", enabled: true, retries: 2, want: []int{24, 27, 30}},
		{name: "affinity hit", enabled: true, binding: 27, retries: 2, want: []int{27, 24, 27}},
		{name: "no retry budget", enabled: true, binding: 27, want: []int{27}},
		{name: "one retry", enabled: true, binding: 27, retries: 1, want: []int{27, 24}},
		{name: "skip retry on failure", enabled: true, skipRetry: true, binding: 27, retries: 2, want: []int{27}},
		{name: "auto group", enabled: true, binding: 27, retries: 1, autoGroups: []string{"default", "vip"}, want: []int{27, 24}},
		{name: "cross group", enabled: true, binding: 27, retries: 1, autoGroups: []string{"default", "vip"}, crossGroup: true, want: []int{27, 24, 34, 37}},
		{name: "skip empty group", enabled: true, binding: 27, retries: 1, autoGroups: []string{"empty", "default"}, want: []int{27, 24, 27}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			common.RetryTimes = tc.retries
			*affinity = operation_setting.ChannelAffinitySetting{
				Enabled: tc.enabled, DefaultTTLSeconds: 60, MaxEntries: 100,
				Rules: []operation_setting.ChannelAffinityRule{{
					Name: t.Name(), ModelRegex: []string{"^" + modelName + "$"}, SkipRetryOnFailure: tc.skipRetry,
					KeySources: []operation_setting.ChannelAffinityKeySource{{Type: "request_header", Key: "X-Session"}},
				}},
			}
			newRequest := func() *gin.Context {
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
				ctx.Request = httptest.NewRequest(http.MethodPost, requestPath, nil)
				ctx.Request.Header.Set("X-Session", t.Name())
				common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
				if tc.autoGroups != nil {
					common.SetContextKey(ctx, constant.ContextKeyTokenAutoGroups, tc.autoGroups)
					common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, tc.crossGroup)
				}
				return ctx
			}
			group := "default"
			if tc.autoGroups != nil {
				group = "auto"
			}
			if tc.binding != 0 {
				previous := newRequest()
				_, found := GetPreferredChannelByAffinity(previous, modelName, group)
				require.False(t, found)
				RecordChannelAffinity(previous, tc.binding)
				t.Cleanup(func() { ClearCurrentChannelAffinityCache(previous) })
			}

			// Attempt 0 follows the distributor: an affinity hit uses the bound
			// channel in the first group that serves it, otherwise normal selection.
			ctx := newRequest()
			var channel *model.Channel
			if preferredID, found := GetPreferredChannelByAffinity(ctx, modelName, group); found {
				affinityGroups := []string{group}
				if group == "auto" {
					affinityGroups = GetRequestAutoGroups(ctx, "default")
				}
				for _, g := range affinityGroups {
					if model.IsChannelEnabledForGroupModel(g, modelName, preferredID) {
						var err error
						channel, err = model.CacheGetChannel(preferredID)
						require.NoError(t, err)
						MarkChannelAffinityUsed(ctx, g, preferredID)
						break
					}
				}
				require.NotNil(t, channel)
			}
			if channel == nil {
				var err error
				channel, _, err = CacheGetRandomSatisfiedChannel(&RetryParam{Ctx: ctx, TokenGroup: group, ModelName: modelName, RequestPath: requestPath})
				require.NoError(t, err)
				require.NotNil(t, channel)
			}

			// The relay loop uses its own RetryParam and re-selects from retry 1.
			retry := &RetryParam{Ctx: ctx, TokenGroup: group, ModelName: modelName, RequestPath: requestPath, Retry: common.GetPointer(0)}
			var attempts []int
			for ; retry.GetRetry() <= common.RetryTimes; retry.IncreaseRetry() {
				if len(attempts) > 0 {
					var err error
					channel, _, err = CacheGetRandomSatisfiedChannel(retry)
					require.NoError(t, err)
					if channel == nil {
						break
					}
				}
				attempts = append(attempts, channel.Id)
				require.LessOrEqual(t, len(attempts), len(tc.want), "retry budget must not increase")
				// Every attempt fails with a retryable upstream error.
				if ShouldSkipRetryAfterChannelAffinityFailure(ctx) || common.RetryTimes-retry.GetRetry() <= 0 {
					break
				}
			}
			assert.Equal(t, tc.want, attempts)
		})
	}
}
