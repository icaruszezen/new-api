package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveIQCheckResults(t *testing.T, checkId string, count int, historyLimit int) []int64 {
	t.Helper()
	ids := make([]int64, 0, count)
	for index := 0; index < count; index++ {
		result := &ModelIQCheckResult{
			CheckId:   checkId,
			CheckedAt: int64(1_700_000_000 + index),
			Status:    ModelIQCheckStatusSuccess,
			ModelName: "gpt-5",
			Prompt:    "1+1=?",
			Content:   "2",
		}
		require.NoError(t, SaveModelIQCheckResult(result, historyLimit))
		require.NotZero(t, result.Id)
		ids = append(ids, result.Id)
	}
	return ids
}

func TestSaveModelIQCheckResultKeepsNewestRowsPerCheck(t *testing.T) {
	truncateTables(t)

	otherIds := saveIQCheckResults(t, "other", 2, 10)
	ids := saveIQCheckResults(t, "c1", 5, 3)

	history, err := GetModelIQCheckHistory([]string{"c1", "other"})
	require.NoError(t, err)

	var c1Ids, keptOtherIds []int64
	for _, row := range history {
		switch row.CheckId {
		case "c1":
			c1Ids = append(c1Ids, row.Id)
		case "other":
			keptOtherIds = append(keptOtherIds, row.Id)
		}
	}
	assert.Equal(t, []int64{ids[4], ids[3], ids[2]}, c1Ids, "only the newest historyLimit rows survive, newest first")
	assert.Equal(t, []int64{otherIds[1], otherIds[0]}, keptOtherIds, "trimming one check must not touch another check's history")
}

func TestSaveModelIQCheckResultShrinksHistoryWhenLimitIsLowered(t *testing.T) {
	truncateTables(t)

	saveIQCheckResults(t, "c1", 4, 10)
	ids := saveIQCheckResults(t, "c1", 1, 1)

	history, err := GetModelIQCheckHistory([]string{"c1"})
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, ids[0], history[0].Id)
}

func TestGetModelIQCheckHistoryReturnsMetadataOnly(t *testing.T) {
	truncateTables(t)

	require.NoError(t, SaveModelIQCheckResult(&ModelIQCheckResult{
		CheckId:          "c1",
		CheckedAt:        1_700_000_000,
		Status:           ModelIQCheckStatusFailed,
		Prompt:           "a long prompt",
		Content:          "<html>large page</html>",
		Reasoning:        "long reasoning",
		ErrorMessage:     "timed out after 300 seconds",
		DurationMs:       300_000,
		CompletionTokens: 77,
	}, 10))

	history, err := GetModelIQCheckHistory([]string{"c1"})
	require.NoError(t, err)
	require.Len(t, history, 1)
	row := history[0]
	assert.Equal(t, "c1", row.CheckId)
	assert.Equal(t, int64(1_700_000_000), row.CheckedAt)
	assert.Equal(t, ModelIQCheckStatusFailed, row.Status)
	assert.Equal(t, int64(300_000), row.DurationMs)
	assert.Equal(t, 77, row.CompletionTokens)
	assert.Equal(t, "timed out after 300 seconds", row.ErrorMessage)
	// The status page polls history for every card; large fields are loaded
	// only on demand through GetModelIQCheckResultsByIds.
	assert.Empty(t, row.Content)
	assert.Empty(t, row.Prompt)
	assert.Empty(t, row.Reasoning)
}

func TestGetModelIQCheckResultsByIdsFiltersByAllowedChecks(t *testing.T) {
	truncateTables(t)

	allowed := saveIQCheckResults(t, "enabled", 2, 10)
	hidden := saveIQCheckResults(t, "disabled", 1, 10)

	rows, err := GetModelIQCheckResultsByIds([]int64{allowed[0], allowed[1], hidden[0], 999_999}, []string{"enabled"})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, row := range rows {
		assert.Equal(t, "enabled", row.CheckId)
		assert.Equal(t, "2", row.Content, "result details carry the full content")
	}

	rows, err = GetModelIQCheckResultsByIds([]int64{hidden[0]}, nil)
	require.NoError(t, err)
	assert.Empty(t, rows, "no allowed checks means nothing is readable")
}

func TestDeleteModelIQCheckDataExcept(t *testing.T) {
	truncateTables(t)

	saveIQCheckResults(t, "keep", 1, 10)
	saveIQCheckResults(t, "drop", 2, 10)
	require.NoError(t, MarkModelIQCheckStarted("keep", 100))
	require.NoError(t, MarkModelIQCheckStarted("drop", 100))

	require.NoError(t, DeleteModelIQCheckDataExcept([]string{"keep"}))

	history, err := GetModelIQCheckHistory([]string{"keep", "drop"})
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "keep", history[0].CheckId)
	states, err := GetModelIQCheckStates([]string{"keep", "drop"})
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, "keep", states[0].CheckId)

	// An empty list means every check was removed from the configuration.
	require.NoError(t, DeleteModelIQCheckDataExcept(nil))

	history, err = GetModelIQCheckHistory([]string{"keep"})
	require.NoError(t, err)
	assert.Empty(t, history)
	states, err = GetModelIQCheckStates([]string{"keep"})
	require.NoError(t, err)
	assert.Empty(t, states)
}

func TestModelIQCheckStateUpsertsTouchOnlyTheirColumn(t *testing.T) {
	truncateTables(t)

	require.NoError(t, RequestModelIQCheckRun("c1", 50))
	require.NoError(t, MarkModelIQCheckStarted("c1", 100))
	require.NoError(t, MarkModelIQCheckFinished("c1", 160))
	require.NoError(t, RequestModelIQCheckRun("c1", 170))
	require.NoError(t, MarkModelIQCheckStarted("c1", 180))

	states, err := GetModelIQCheckStates([]string{"c1"})
	require.NoError(t, err)
	require.Len(t, states, 1)
	assert.Equal(t, ModelIQCheckState{
		CheckId:        "c1",
		LastStartedAt:  180,
		LastFinishedAt: 160,
		RunRequestedAt: 170,
	}, states[0])
}
