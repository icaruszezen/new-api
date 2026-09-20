package channelmonitor

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/stretchr/testify/assert"
)

func TestShouldIgnoreMonitorError(t *testing.T) {
	cases := []struct {
		name   string
		err    *types.NewAPIError
		expect bool
	}{
		{name: "nil error is scored", err: nil, expect: false},
		{
			name:   "quota errors are ignored",
			err:    types.NewError(errors.New("insufficient quota"), types.ErrorCodeInsufficientUserQuota),
			expect: true,
		},
		{
			name:   "pre-consume failures are ignored",
			err:    types.NewError(errors.New("pre-consume failed"), types.ErrorCodePreConsumeTokenQuotaFailed),
			expect: true,
		},
		{
			name:   "content policy is ignored",
			err:    types.NewError(errors.New("prompt blocked"), types.ErrorCodePromptBlocked),
			expect: true,
		},
		{
			name:   "model not found is ignored",
			err:    types.NewError(errors.New("unknown model"), types.ErrorCodeModelNotFound),
			expect: true,
		},
		{
			name:   "group access is ignored",
			err:    types.NewError(errors.New("group not allowed"), types.ErrorCodeAccessDenied),
			expect: true,
		},
		{
			name: "client cancel is ignored",
			err: types.NewError(
				context.Canceled,
				types.ErrorCodeBadResponse,
			),
			expect: true,
		},
		{
			name:   "401 is ignored",
			err:    types.NewErrorWithStatusCode(errors.New("unauthorized"), types.ErrorCodeInvalidRequest, http.StatusUnauthorized),
			expect: true,
		},
		{
			name:   "sensitive words are ignored",
			err:    types.NewError(errors.New("blocked keyword"), types.ErrorCodeSensitiveWordsDetected),
			expect: true,
		},
		{
			name:   "csam policy is ignored",
			err:    types.NewError(errors.New("csam"), types.ErrorCodeViolationFeeGrokCSAM),
			expect: true,
		},
		{
			name:   "client disconnect 499 is ignored",
			err:    types.NewErrorWithStatusCode(errors.New("client closed"), types.ErrorCodeBadResponse, 499),
			expect: true,
		},
		{
			name:   "context window overflow is ignored",
			err:    types.NewError(errors.New("maximum context length exceeded"), types.ErrorCodeInvalidRequest),
			expect: true,
		},
		{
			name:   "timeout stays a failure",
			err:    types.NewError(errors.New("gateway timeout"), types.ErrorCodeDoRequestFailed),
			expect: false,
		},
		{
			name:   "429 stays a failure",
			err:    types.NewErrorWithStatusCode(errors.New("rate limit"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests),
			expect: false,
		},
		{
			name:   "upstream 5xx stays a failure",
			err:    types.NewErrorWithStatusCode(errors.New("bad gateway"), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway),
			expect: false,
		},
		{
			name:   "channel-prefixed errors stay a failure",
			err:    types.NewError(errors.New("no key"), types.ErrorCodeChannelNoAvailableKey),
			expect: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expect, shouldIgnoreMonitorError(testCase.err))
		})
	}
}
