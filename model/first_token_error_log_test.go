package model

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupFirstTokenErrorLogTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&FirstTokenErrorLog{}, &FirstTokenErrorBody{}))
	prevDB := DB
	DB = db
	t.Cleanup(func() {
		DB = prevDB
	})
}

func TestPrepareFirstTokenErrorBodyTruncates(t *testing.T) {
	small := []byte(`{"model":"gpt","input":"hi"}`)
	body, truncated, size := PrepareFirstTokenErrorBody(small)
	assert.False(t, truncated)
	assert.Equal(t, string(small), body)
	assert.Equal(t, len(small), size)

	raw := []byte(`{"model":"gpt","input":"` + strings.Repeat("x", common.FirstTokenErrorBodyMaxBytes+8) + `"}`)
	body, truncated, size = PrepareFirstTokenErrorBody(raw)
	assert.True(t, truncated)
	assert.LessOrEqual(t, size, common.FirstTokenErrorBodyMaxBytes)
	assert.Contains(t, body, `"model"`)
}

func TestTruncateUTF8BytesDoesNotSplitRune(t *testing.T) {
	raw := []byte("ab你好")
	require.Greater(t, len(raw), 4)
	truncated := truncateUTF8Bytes(raw, 4)
	assert.True(t, utf8.ValidString(truncated))
	assert.Equal(t, "ab", truncated)
	assert.Equal(t, string(raw), truncateUTF8Bytes(raw, len(raw)))
}

func TestFirstTokenErrorLogMaxKeepPrune(t *testing.T) {
	setupFirstTokenErrorLogTestDB(t)
	old := common.FirstTokenErrorLogMaxKeep
	common.FirstTokenErrorLogMaxKeep = 3
	t.Cleanup(func() {
		common.FirstTokenErrorLogMaxKeep = old
	})

	for i := 0; i < 4; i++ {
		log := &FirstTokenErrorLog{RequestId: "r", Username: "u", CompletionTokens: 1}
		require.NoError(t, InsertFirstTokenErrorLog(log))
		require.NoError(t, SaveFirstTokenErrorBody(log.Id, `{"n":1}`, false, 7))
	}
	require.NoError(t, PruneFirstTokenErrorLogs())
	var n int64
	require.NoError(t, DB.Model(&FirstTokenErrorLog{}).Count(&n).Error)
	assert.Equal(t, int64(3), n)
	var bodies int64
	require.NoError(t, DB.Model(&FirstTokenErrorBody{}).Count(&bodies).Error)
	assert.Equal(t, int64(3), bodies)

	common.FirstTokenErrorLogMaxKeep = 1
	require.NoError(t, PruneFirstTokenErrorLogs())
	require.NoError(t, DB.Model(&FirstTokenErrorLog{}).Count(&n).Error)
	assert.Equal(t, int64(1), n)
}

func TestValidateFirstTokenErrorLogMaxKeep(t *testing.T) {
	assert.Error(t, validateFirstTokenErrorLogMaxKeep("0"))
	assert.Error(t, validateFirstTokenErrorLogMaxKeep("x"))
	assert.Error(t, validateFirstTokenErrorLogMaxKeep("100001"))
	assert.NoError(t, validateFirstTokenErrorLogMaxKeep("5000"))
}
