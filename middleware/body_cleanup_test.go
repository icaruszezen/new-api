package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type closeCountingBodyStorage struct {
	common.BodyStorage
	closeCalls int
}

func (storage *closeCountingBodyStorage) Close() error {
	storage.closeCalls++
	return storage.BodyStorage.Close()
}

// Relay routes have no recovery of their own; a handler panic unwinds to the
// global recovery. The cached request body (a temp file plus disk-cache
// accounting when it is disk-backed) must still be released.
func TestBodyStorageCleanupReleasesBodyWhenHandlerPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		panics     bool
		wantStatus int
	}{
		{name: "handler returns", wantStatus: http.StatusOK},
		{name: "handler panics", panics: true, wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := common.CreateBodyStorage([]byte(`{"model":"gpt-4o"}`))
			require.NoError(t, err)
			tracked := &closeCountingBodyStorage{BodyStorage: storage}

			engine := gin.New()
			engine.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
				c.AbortWithStatus(http.StatusInternalServerError)
			}))
			engine.Use(BodyStorageCleanup())
			engine.POST("/v1/chat/completions", func(c *gin.Context) {
				c.Set(common.KeyBodyStorage, tracked)
				if tt.panics {
					panic("adaptor bug")
				}
				c.Status(http.StatusOK)
			})
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))

			require.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, 1, tracked.closeCalls)
		})
	}
}
