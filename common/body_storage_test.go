package common

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReplayableBodyReaderKeepsStorageLifecycleWithCaller(t *testing.T) {
	payload := []byte(`{"model":"test-model","input":"hello"}`)
	storage, err := CreateBodyStorage(payload)
	require.NoError(t, err)
	defer storage.Close()

	body := NewReplayableBodyReader(storage)
	assert.EqualValues(t, len(payload), body.Size())
	_, exposesCloser := any(body).(io.Closer)
	assert.False(t, exposesCloser, "the request body must not expose the storage closer")

	req, err := http.NewRequest(http.MethodPost, "https://example.com", body)
	require.NoError(t, err)
	require.NoError(t, req.Body.Close())

	replayBody, err := body.NewReader()
	require.NoError(t, err, "closing the HTTP request body must not close the storage")
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, payload, replay)

	require.NoError(t, storage.Close())
	_, err = body.NewReader()
	require.ErrorIs(t, err, ErrStorageClosed)
}

// A same-channel retry sends the stored body again while the previous
// attempt's transport may still be reading it, and GetBodyStorage seeks the
// shared storage in between. Each replayable reader must therefore keep its
// own offset, for memory and disk storage alike.
func TestReplayableBodyReadersKeepIndependentOffsets(t *testing.T) {
	payload := []byte(`{"model":"test-model","messages":[{"role":"user","content":"abcdefghijklmnopqrstuvwxyz0123456789"}]}`)

	tests := []struct {
		name string
		disk bool
	}{
		{name: "memory storage", disk: false},
		{name: "disk storage", disk: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.disk {
				prev := GetDiskCacheConfig()
				SetDiskCacheConfig(DiskCacheConfig{Enabled: true, ThresholdMB: 0, MaxSizeMB: 64, Path: t.TempDir()})
				t.Cleanup(func() { SetDiskCacheConfig(prev) })
			}
			storage, err := CreateBodyStorage(payload)
			require.NoError(t, err)
			t.Cleanup(func() { _ = storage.Close() })
			require.Equal(t, tt.disk, storage.IsDisk())

			first := NewReplayableBodyReader(storage)
			second := NewReplayableBodyReader(storage)
			var firstGot, secondGot []byte
			chunk := make([]byte, 7)
			var firstErr, secondErr error
			// Bounded so a reader that never reaches EOF fails instead of hanging.
			maxReads := len(payload)/len(chunk) + 2
			for i := 0; i < maxReads && (firstErr == nil || secondErr == nil); i++ {
				if firstErr == nil {
					var n int
					n, firstErr = first.Read(chunk)
					firstGot = append(firstGot, chunk[:n]...)
				}
				// What GetBodyStorage and a direct storage read do between attempts.
				_, err := storage.Seek(0, io.SeekStart)
				require.NoError(t, err)
				_, err = storage.Read(make([]byte, 5))
				require.NoError(t, err)
				if secondErr == nil {
					var n int
					n, secondErr = second.Read(chunk)
					secondGot = append(secondGot, chunk[:n]...)
				}
			}

			require.ErrorIs(t, firstErr, io.EOF)
			require.ErrorIs(t, secondErr, io.EOF)
			assert.Equal(t, payload, firstGot)
			assert.Equal(t, payload, secondGot)

			// A reader created while the shared cursor sits at the end still
			// starts at the beginning of the payload.
			_, err = storage.Seek(0, io.SeekEnd)
			require.NoError(t, err)
			replay, err := io.ReadAll(NewReplayableBodyReader(storage))
			require.NoError(t, err)
			assert.Equal(t, payload, replay)
		})
	}
}
