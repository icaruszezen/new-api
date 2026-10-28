package aws

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	relaytypes "github.com/QuantumNous/new-api/relaykit/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream/eventstreamapi"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const awsTestModel = "anthropic.claude-3-5-sonnet-20240620-v1:0"

// A Claude message as Bedrock streams it: the start carries the input usage
// and the client-visible "partial" text, the finish carries the final usage.
var (
	awsClaudeStreamStart = []string{
		`{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","model":"claude-test","content":[],"usage":{"input_tokens":100,"output_tokens":1}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}`,
	}
	awsClaudeStreamFinish = []string{
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":423}}`,
		`{"type":"message_stop"}`,
	}
)

const awsClaudePingEvent = `{"type":"ping"}`

type awsHTTPClientFunc func(*http.Request) (*http.Response, error)

func (f awsHTTPClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

type awsNotifyingResponseWriter struct {
	*httptest.ResponseRecorder
	notifyOn []byte
	notified chan int
	// hold, when set, keeps the Flush that delivered notifyOn blocked until it
	// is closed, like a client slow to take that chunk.
	hold chan struct{}
	once sync.Once
}

func newAwsNotifyingResponseWriter(notifyOn string) *awsNotifyingResponseWriter {
	return &awsNotifyingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		notifyOn:         []byte(notifyOn),
		notified:         make(chan int, 1),
	}
}

func (w *awsNotifyingResponseWriter) Write(data []byte) (int, error) {
	return w.ResponseRecorder.Write(data)
}

func (w *awsNotifyingResponseWriter) Flush() {
	w.ResponseRecorder.Flush()
	if bytes.Contains(w.Body.Bytes(), w.notifyOn) {
		w.once.Do(func() {
			w.notified <- w.Body.Len()
			if w.hold != nil {
				<-w.hold
			}
		})
	}
}

func newAwsTestClient(httpClient bedrockruntime.HTTPClient) *bedrockruntime.Client {
	return bedrockruntime.New(bedrockruntime.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("https://bedrock.test"),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			"access-key", "secret-key", "",
		)),
		HTTPClient: httpClient,
		Retryer:    aws.NopRetryer{},
	})
}

func newAwsTestContext(writer http.ResponseWriter, requestContext context.Context) *gin.Context {
	c, _ := gin.CreateTestContext(writer)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).WithContext(requestContext)
	return c
}

func newAwsTestRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		StartTime:          time.Now(),
		IsStream:           true,
		OriginModelName:    awsTestModel,
		RelayFormat:        relaytypes.RelayFormatOpenAI,
		ShouldIncludeUsage: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: awsTestModel,
		},
	}
}

func newAwsInvokeModelInput() *bedrockruntime.InvokeModelInput {
	return &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(awsTestModel),
		Body:        []byte(`{}`),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}
}

func newAwsStreamInput() *bedrockruntime.InvokeModelWithResponseStreamInput {
	return &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(awsTestModel),
		Body:        []byte(`{}`),
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	}
}

// useAwsStreamLimits pins the globals that bound an SDK call: no RELAY_TIMEOUT,
// and the given STREAMING_TIMEOUT and STREAM_DRAIN_MAX_SECONDS.
func useAwsStreamLimits(t *testing.T, streamingTimeout int, drainMaxSeconds int) {
	t.Helper()
	originalRelayTimeout := common.RelayTimeout
	originalStreamingTimeout := constant.StreamingTimeout
	originalDrainMaxSeconds := constant.StreamDrainMaxSeconds
	common.RelayTimeout = 0
	constant.StreamingTimeout = streamingTimeout
	constant.StreamDrainMaxSeconds = drainMaxSeconds
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
		constant.StreamingTimeout = originalStreamingTimeout
		constant.StreamDrainMaxSeconds = originalDrainMaxSeconds
	})
}

func writeAwsStreamEvent(writer io.Writer, data string) error {
	payload, err := common.Marshal(struct {
		Bytes []byte `json:"bytes"`
	}{Bytes: []byte(data)})
	if err != nil {
		return err
	}

	return eventstream.NewEncoder().Encode(writer, eventstream.Message{
		Headers: eventstream.Headers{
			{Name: eventstreamapi.MessageTypeHeader, Value: eventstream.StringValue(eventstreamapi.EventMessageType)},
			{Name: eventstreamapi.EventTypeHeader, Value: eventstream.StringValue("chunk")},
			{Name: eventstreamapi.ContentTypeHeader, Value: eventstream.StringValue("application/json")},
		},
		Payload: payload,
	})
}

// writeAwsStreamException writes the exception Bedrock sends in place of the
// rest of the event stream when the model fails mid-stream.
func writeAwsStreamException(writer io.Writer, message string) error {
	payload, err := common.Marshal(map[string]string{"message": message})
	if err != nil {
		return err
	}

	return eventstream.NewEncoder().Encode(writer, eventstream.Message{
		Headers: eventstream.Headers{
			{Name: eventstreamapi.MessageTypeHeader, Value: eventstream.StringValue(eventstreamapi.ExceptionMessageType)},
			{Name: eventstreamapi.ExceptionTypeHeader, Value: eventstream.StringValue("modelStreamErrorException")},
			{Name: eventstreamapi.ContentTypeHeader, Value: eventstream.StringValue("application/json")},
		},
		Payload: payload,
	})
}

func newAwsStreamResponse(request *http.Request, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header: http.Header{
			"Content-Type":                []string{"application/vnd.amazon.eventstream"},
			"X-Amzn-Bedrock-Content-Type": []string{"application/json"},
		},
		Body:    body,
		Request: request,
	}
}

func newAwsJSONResponse(request *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Request:    request,
	}
}

type awsHandlerResult struct {
	err   *relaytypes.NewAPIError
	usage *dto.Usage
}

func runAwsHandler(handle func(*gin.Context, *relaycommon.RelayInfo, *Adaptor) (*relaytypes.NewAPIError, *dto.Usage), c *gin.Context, info *relaycommon.RelayInfo, adaptor *Adaptor) <-chan awsHandlerResult {
	results := make(chan awsHandlerResult, 1)
	go func() {
		err, usage := handle(c, info, adaptor)
		results <- awsHandlerResult{err: err, usage: usage}
	}()
	return results
}

func waitAwsHandler(t *testing.T, results <-chan awsHandlerResult, within time.Duration) awsHandlerResult {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(within):
		t.Fatal("AWS handler did not return")
		return awsHandlerResult{}
	}
}

// awsStreamRun is awsStreamHandler relaying a Bedrock event stream the test
// writes by hand. The client has received the start of the message, up to the
// "partial" text, and nothing has been written to it since.
type awsStreamRun struct {
	info             *relaycommon.RelayInfo
	client           *awsNotifyingResponseWriter
	bodyLenAtPartial int
	disconnect       context.CancelFunc
	upstreamContext  context.Context
	upstream         *io.PipeWriter
	results          <-chan awsHandlerResult
}

func startAwsStreamRun(t *testing.T, drain bool) *awsStreamRun {
	t.Helper()
	requestContext, disconnect := context.WithCancel(context.Background())
	t.Cleanup(disconnect)

	type upstreamCall struct {
		ctx    context.Context
		writer *io.PipeWriter
	}
	calls := make(chan upstreamCall, 1)
	client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		reader, writer := io.Pipe()
		calls <- upstreamCall{ctx: request.Context(), writer: writer}
		return newAwsStreamResponse(request, reader), nil
	}))
	responseWriter := newAwsNotifyingResponseWriter("partial")
	info := newAwsTestRelayInfo()
	info.DrainUpstreamOnClientDisconnect = drain
	adaptor := &Adaptor{AwsClient: client, AwsReq: newAwsStreamInput()}
	results := runAwsHandler(awsStreamHandler, newAwsTestContext(responseWriter, requestContext), info, adaptor)

	var call upstreamCall
	select {
	case call = <-calls:
	case <-time.After(5 * time.Second):
		t.Fatal("AWS stream request did not start")
	}
	t.Cleanup(func() { _ = call.writer.Close() })

	for _, event := range awsClaudeStreamStart {
		require.NoError(t, writeAwsStreamEvent(call.writer, event))
	}
	var bodyLenAtPartial int
	select {
	case bodyLenAtPartial = <-responseWriter.notified:
	case <-time.After(5 * time.Second):
		t.Fatal("partial response was not written")
	}

	return &awsStreamRun{
		info:             info,
		client:           responseWriter,
		bodyLenAtPartial: bodyLenAtPartial,
		disconnect:       disconnect,
		upstreamContext:  call.ctx,
		upstream:         call.writer,
		results:          results,
	}
}

func TestDoAwsClientRequest_AppliesRuntimeHeaderOverrideToAnthropicBeta(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName:           "claude-3-5-sonnet-20240620",
		IsStream:                  false,
		UseRuntimeHeadersOverride: true,
		RuntimeHeadersOverride: map[string]any{
			"anthropic-beta": "computer-use-2025-01-24",
		},
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:            "access-key|secret-key|us-east-1",
			UpstreamModelName: "claude-3-5-sonnet-20240620",
		},
	}

	requestBody := bytes.NewBufferString(`{"messages":[{"role":"user","content":"hello"}],"max_tokens":128}`)
	adaptor := &Adaptor{}

	_, err := doAwsClientRequest(ctx, info, adaptor, requestBody)
	require.NoError(t, err)

	awsReq, ok := adaptor.AwsReq.(*bedrockruntime.InvokeModelInput)
	require.True(t, ok)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(awsReq.Body, &payload))

	anthropicBeta, exists := payload["anthropic_beta"]
	require.True(t, exists)

	values, ok := anthropicBeta.([]any)
	require.True(t, ok)
	require.Equal(t, []any{"computer-use-2025-01-24"}, values)
}

func TestNewAwsInvokeContextFollowsClientUnlessDraining(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	tests := []struct {
		name                 string
		relayTimeout         int
		drain                bool
		wantDeadline         bool
		wantCanceledByClient bool
	}{
		{name: "without relay timeout", relayTimeout: 0, wantDeadline: false, wantCanceledByClient: true},
		{name: "with relay timeout", relayTimeout: 30, wantDeadline: true, wantCanceledByClient: true},
		{name: "draining without relay timeout", relayTimeout: 0, drain: true, wantDeadline: false, wantCanceledByClient: false},
		{name: "draining keeps relay timeout", relayTimeout: 30, drain: true, wantDeadline: true, wantCanceledByClient: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common.RelayTimeout = test.relayTimeout
			parent, cancelParent := context.WithCancel(context.Background())
			invokeContext, cancelInvoke := newAwsInvokeContext(parent, test.drain)
			defer cancelInvoke()

			_, hasDeadline := invokeContext.Deadline()
			assert.Equal(t, test.wantDeadline, hasDeadline)

			cancelParent()
			assert.Equal(t, test.wantCanceledByClient, invokeContext.Err() != nil)

			// The handler can always end the call itself.
			cancelInvoke()
			require.ErrorIs(t, invokeContext.Err(), context.Canceled)
		})
	}
}

func TestNewAwsInvokeErrorSkipsRetryOnlyForClientCancellation(t *testing.T) {
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name           string
		requestContext context.Context
		err            error
		wantSkipRetry  bool
	}{
		{
			name:           "client context canceled",
			requestContext: canceledContext,
			err:            context.Canceled,
			wantSkipRetry:  true,
		},
		{
			name:           "relay timeout with live client context",
			requestContext: context.Background(),
			err:            context.DeadlineExceeded,
			wantSkipRetry:  false,
		},
		{
			name:           "upstream error with live client context",
			requestContext: context.Background(),
			err:            errors.New("upstream failed"),
			wantSkipRetry:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := newAwsInvokeError(test.requestContext, test.err, "InvokeModel")
			assert.Equal(t, test.wantSkipRetry, relaytypes.IsSkipRetryError(err))
		})
	}
}

func TestAwsHandlersCancelSdkRequestAndSkipRetry(t *testing.T) {
	useAwsStreamLimits(t, 30, 1800)

	tests := []struct {
		name    string
		request any
		handle  func(*gin.Context, *relaycommon.RelayInfo, *Adaptor) (*relaytypes.NewAPIError, *dto.Usage)
	}{
		{name: "non-stream", request: newAwsInvokeModelInput(), handle: awsHandler},
		{name: "stream", request: newAwsStreamInput(), handle: awsStreamHandler},
		{name: "nova", request: newAwsInvokeModelInput(), handle: handleNovaRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestContext, cancelRequest := context.WithCancel(context.Background())
			t.Cleanup(cancelRequest)

			upstreamContexts := make(chan context.Context, 1)
			client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
				upstreamContexts <- request.Context()
				<-request.Context().Done()
				return nil, request.Context().Err()
			}))
			adaptor := &Adaptor{AwsClient: client, AwsReq: test.request}
			c := newAwsTestContext(httptest.NewRecorder(), requestContext)
			results := runAwsHandler(test.handle, c, newAwsTestRelayInfo(), adaptor)

			var upstreamContext context.Context
			select {
			case upstreamContext = <-upstreamContexts:
			case result := <-results:
				t.Fatalf("handler returned before issuing AWS request: %v", result.err)
			case <-time.After(5 * time.Second):
				t.Fatal("AWS request did not start")
			}

			cancelRequest()
			result := waitAwsHandler(t, results, 5*time.Second)

			require.ErrorIs(t, upstreamContext.Err(), context.Canceled)
			require.NotNil(t, result.err)
			assert.True(t, relaytypes.IsSkipRetryError(result.err))
			assert.Nil(t, result.usage)
		})
	}
}

// With drain on, a client that leaves while Bedrock is still answering does
// not cancel the SDK call, so the request settles on the upstream's usage.
func TestAwsHandlersFinishSdkCallWhenDrainingAfterClientLeaves(t *testing.T) {
	useAwsStreamLimits(t, 30, 1800)

	tests := []struct {
		name                 string
		request              any
		handle               func(*gin.Context, *relaycommon.RelayInfo, *Adaptor) (*relaytypes.NewAPIError, *dto.Usage)
		respond              func(*http.Request) (*http.Response, error)
		wantPromptTokens     int
		wantCompletionTokens int
	}{
		{
			name:    "non-stream",
			request: newAwsInvokeModelInput(),
			handle:  awsHandler,
			respond: func(request *http.Request) (*http.Response, error) {
				return newAwsJSONResponse(request, `{"id":"msg_test","type":"message","role":"assistant","model":"claude-test","content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":423}}`), nil
			},
			wantPromptTokens:     100,
			wantCompletionTokens: 423,
		},
		{
			name:    "stream",
			request: newAwsStreamInput(),
			handle:  awsStreamHandler,
			respond: func(request *http.Request) (*http.Response, error) {
				var body bytes.Buffer
				for _, event := range append(append([]string{}, awsClaudeStreamStart...), awsClaudeStreamFinish...) {
					if err := writeAwsStreamEvent(&body, event); err != nil {
						return nil, err
					}
				}
				return newAwsStreamResponse(request, io.NopCloser(&body)), nil
			},
			wantPromptTokens:     100,
			wantCompletionTokens: 423,
		},
		{
			name:    "nova",
			request: newAwsInvokeModelInput(),
			handle:  handleNovaRequest,
			respond: func(request *http.Request) (*http.Response, error) {
				return newAwsJSONResponse(request, `{"output":{"message":{"content":[{"text":"hi"}]}},"usage":{"inputTokens":5,"outputTokens":7,"totalTokens":12}}`), nil
			},
			wantPromptTokens:     5,
			wantCompletionTokens: 7,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestContext, disconnect := context.WithCancel(context.Background())
			t.Cleanup(disconnect)

			upstreamContexts := make(chan context.Context, 1)
			answer := make(chan struct{})
			client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
				upstreamContexts <- request.Context()
				select {
				case <-answer:
					return test.respond(request)
				case <-request.Context().Done():
					return nil, request.Context().Err()
				}
			}))
			info := newAwsTestRelayInfo()
			info.DrainUpstreamOnClientDisconnect = true
			adaptor := &Adaptor{AwsClient: client, AwsReq: test.request}
			results := runAwsHandler(test.handle, newAwsTestContext(httptest.NewRecorder(), requestContext), info, adaptor)

			var upstreamContext context.Context
			select {
			case upstreamContext = <-upstreamContexts:
			case <-time.After(5 * time.Second):
				t.Fatal("AWS request did not start")
			}

			disconnect()
			require.NoError(t, upstreamContext.Err(), "a client disconnect must not cancel a draining SDK call")
			close(answer)
			result := waitAwsHandler(t, results, 5*time.Second)

			require.Nil(t, result.err)
			require.NotNil(t, result.usage)
			assert.Equal(t, test.wantPromptTokens, result.usage.PromptTokens)
			assert.Equal(t, test.wantCompletionTokens, result.usage.CompletionTokens)
		})
	}
}

func TestAwsStreamHandlerRecordsUpstreamEnd(t *testing.T) {
	useAwsStreamLimits(t, 30, 1800)

	tests := []struct {
		name             string
		events           []string
		exception        string
		wantEndReason    relaycommon.StreamEndReason
		wantEndError     string
		wantHandlerErr   bool
		wantOutputTokens int
		wantDone         bool
	}{
		{
			name:             "event stream ends",
			events:           append(append([]string{}, awsClaudeStreamStart...), awsClaudeStreamFinish...),
			wantEndReason:    relaycommon.StreamEndReasonEOF,
			wantOutputTokens: 423,
			wantDone:         true,
		},
		{
			// The partial message still settles, as a scanner error does on the SSE path.
			name:          "bedrock exception mid-stream",
			events:        awsClaudeStreamStart,
			exception:     "model stream failed",
			wantEndReason: relaycommon.StreamEndReasonScannerErr,
			wantEndError:  "model stream failed",
			wantDone:      true,
		},
		{
			name:           "claude error event",
			events:         append(append([]string{}, awsClaudeStreamStart...), `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`),
			wantEndReason:  relaycommon.StreamEndReasonHandlerStop,
			wantEndError:   "Overloaded",
			wantHandlerErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newAwsTestClient(awsHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
				var body bytes.Buffer
				for _, event := range test.events {
					if err := writeAwsStreamEvent(&body, event); err != nil {
						return nil, err
					}
				}
				if test.exception != "" {
					if err := writeAwsStreamException(&body, test.exception); err != nil {
						return nil, err
					}
				}
				return newAwsStreamResponse(request, io.NopCloser(&body)), nil
			}))
			adaptor := &Adaptor{AwsClient: client, AwsReq: newAwsStreamInput()}
			recorder := httptest.NewRecorder()
			info := newAwsTestRelayInfo()

			handlerErr, usage := awsStreamHandler(newAwsTestContext(recorder, context.Background()), info, adaptor)

			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, test.wantEndReason, info.StreamStatus.EndReason)
			assert.False(t, info.StreamStatus.ClientGone)
			if test.wantEndError == "" {
				assert.NoError(t, info.StreamStatus.EndError)
			} else {
				assert.ErrorContains(t, info.StreamStatus.EndError, test.wantEndError)
			}
			assert.Equal(t, test.wantDone, bytes.Contains(recorder.Body.Bytes(), []byte("[DONE]")))
			if test.wantHandlerErr {
				require.NotNil(t, handlerErr)
				assert.Nil(t, usage)
				return
			}
			require.Nil(t, handlerErr)
			require.NotNil(t, usage)
			require.NotNil(t, usage.BillingUsage)
			require.NotNil(t, usage.BillingUsage.ClaudeUsage)
			assert.Equal(t, 100, usage.BillingUsage.ClaudeUsage.InputTokens)
			if test.wantOutputTokens > 0 {
				assert.Equal(t, test.wantOutputTokens, usage.BillingUsage.ClaudeUsage.OutputTokens)
			}
		})
	}
}

func TestAwsStreamHandlerClosesUpstreamWhenClientLeavesWithoutDrain(t *testing.T) {
	useAwsStreamLimits(t, 30, 1800)
	run := startAwsStreamRun(t, false)

	run.disconnect()
	result := waitAwsHandler(t, run.results, 5*time.Second)

	// The SDK call is cancelled and the event stream closed, so Bedrock stops
	// generating and the final usage never arrives.
	require.ErrorIs(t, run.upstreamContext.Err(), context.Canceled)
	assert.ErrorIs(t, writeAwsStreamEvent(run.upstream, awsClaudeStreamFinish[0]), io.ErrClosedPipe)

	require.Nil(t, result.err)
	require.NotNil(t, result.usage)
	require.NotNil(t, result.usage.BillingUsage)
	require.NotNil(t, result.usage.BillingUsage.ClaudeUsage)
	assert.Equal(t, dto.BillingUsageSourceClaudeMessages, result.usage.BillingUsage.Source)
	assert.Equal(t, dto.BillingUsageSemanticAnthropic, result.usage.BillingUsage.Semantic)
	assert.Equal(t, 100, result.usage.BillingUsage.ClaudeUsage.InputTokens)
	assert.Less(t, result.usage.BillingUsage.ClaudeUsage.OutputTokens, 423)

	require.NotNil(t, run.info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, run.info.StreamStatus.EndReason)
	assert.True(t, run.info.StreamStatus.ClientGone)
	assert.Equal(t, run.bodyLenAtPartial, run.client.Body.Len())
	assert.NotContains(t, run.client.Body.String(), "[DONE]")
}

// Over a real HTTP transport the disconnect also aborts the SDK's body read, so
// a handler still busy with the previous chunk finds both the disconnect and
// the failed event stream; it must record the client leaving, not an upstream
// error.
func TestAwsStreamHandlerRecordsClientGoneOverHTTPTransport(t *testing.T) {
	useAwsStreamLimits(t, 30, 1800)

	upstreamClosed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		w.Header().Set("X-Amzn-Bedrock-Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		for _, event := range awsClaudeStreamStart {
			if err := writeAwsStreamEvent(w, event); err != nil {
				return
			}
		}
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(upstreamClosed)
	}))
	t.Cleanup(server.Close)
	requestContext, disconnect := context.WithCancel(context.Background())
	t.Cleanup(disconnect)

	client := bedrockruntime.New(bedrockruntime.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(server.URL),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			"access-key", "secret-key", "",
		)),
		HTTPClient: server.Client(),
		Retryer:    aws.NopRetryer{},
	})
	responseWriter := newAwsNotifyingResponseWriter("partial")
	responseWriter.hold = make(chan struct{})
	info := newAwsTestRelayInfo()
	adaptor := &Adaptor{AwsClient: client, AwsReq: newAwsStreamInput()}
	results := runAwsHandler(awsStreamHandler, newAwsTestContext(responseWriter, requestContext), info, adaptor)

	select {
	case <-responseWriter.notified:
	case <-time.After(5 * time.Second):
		t.Fatal("partial response was not written")
	}
	disconnect()
	select {
	case <-upstreamClosed:
	case <-time.After(5 * time.Second):
		t.Fatal("Bedrock connection was not closed after the client disconnected")
	}
	close(responseWriter.hold)
	result := waitAwsHandler(t, results, 5*time.Second)

	require.Nil(t, result.err)
	require.NotNil(t, info.StreamStatus)
	assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	assert.ErrorIs(t, info.StreamStatus.EndError, context.Canceled)
	assert.True(t, info.StreamStatus.ClientGone)
}

func TestAwsStreamHandlerDrainsUpstreamAfterClientLeaves(t *testing.T) {
	useAwsStreamLimits(t, 1, 1800)
	run := startAwsStreamRun(t, true)

	run.disconnect()
	select {
	case <-run.results:
		t.Fatal("handler returned right after the client disconnected")
	case <-time.After(200 * time.Millisecond):
	}
	require.NoError(t, run.upstreamContext.Err(), "a client disconnect must not cancel a draining SDK call")

	// An upstream that only pings for longer than STREAMING_TIMEOUT is alive:
	// every event resets the idle watchdog.
	for i := 0; i < 6; i++ {
		time.Sleep(200 * time.Millisecond)
		require.NoError(t, writeAwsStreamEvent(run.upstream, awsClaudePingEvent), "drain must not close an upstream that keeps sending events")
	}
	for _, event := range awsClaudeStreamFinish {
		require.NoError(t, writeAwsStreamEvent(run.upstream, event))
	}
	require.NoError(t, run.upstream.Close())
	result := waitAwsHandler(t, run.results, 5*time.Second)

	require.Nil(t, result.err)
	require.NotNil(t, result.usage)
	require.NotNil(t, result.usage.BillingUsage)
	require.NotNil(t, result.usage.BillingUsage.ClaudeUsage)
	assert.Equal(t, 100, result.usage.BillingUsage.ClaudeUsage.InputTokens)
	assert.Equal(t, 423, result.usage.BillingUsage.ClaudeUsage.OutputTokens)

	status := run.info.StreamStatus
	require.NotNil(t, status)
	assert.Equal(t, relaycommon.StreamEndReasonDrained, status.EndReason)
	assert.Equal(t, relaycommon.StreamEndReasonEOF, status.UpstreamEndReason())
	assert.True(t, status.ClientGone)
	assert.Positive(t, status.DrainDuration)
	assert.False(t, status.HasErrors())
	// Nothing more reaches the gone client, the final usage chunk and [DONE] included.
	assert.Equal(t, run.bodyLenAtPartial, run.client.Body.Len())
}

func TestAwsStreamHandlerBoundsIdleAndDrainTime(t *testing.T) {
	tests := []struct {
		name             string
		drain            bool
		disconnect       bool
		streamingTimeout int
		drainMaxSeconds  int
		keepAlive        bool
		wantEndReason    relaycommon.StreamEndReason
		wantErrors       []string
	}{
		{
			name:             "connected upstream goes idle",
			streamingTimeout: 1,
			drainMaxSeconds:  1800,
			wantEndReason:    relaycommon.StreamEndReasonTimeout,
		},
		{
			name:             "drained upstream goes idle",
			drain:            true,
			disconnect:       true,
			streamingTimeout: 1,
			drainMaxSeconds:  1800,
			wantEndReason:    relaycommon.StreamEndReasonClientGone,
			wantErrors:       []string{"drain idle timeout"},
		},
		{
			// Keepalives alone must not hold a drain open forever.
			name:             "drain reaches its cap",
			drain:            true,
			disconnect:       true,
			streamingTimeout: 30,
			drainMaxSeconds:  1,
			keepAlive:        true,
			wantEndReason:    relaycommon.StreamEndReasonClientGone,
			wantErrors:       []string{"drain max duration reached"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useAwsStreamLimits(t, test.streamingTimeout, test.drainMaxSeconds)
			run := startAwsStreamRun(t, test.drain)
			if test.disconnect {
				run.disconnect()
			}

			deadline := time.After(3 * time.Second)
			keepAlive := time.NewTicker(100 * time.Millisecond)
			defer keepAlive.Stop()
			var result awsHandlerResult
		wait:
			for {
				select {
				case result = <-run.results:
					break wait
				case <-keepAlive.C:
					if test.keepAlive {
						_ = writeAwsStreamEvent(run.upstream, awsClaudePingEvent)
					}
				case <-deadline:
					t.Fatal("handler did not end the stream")
				}
			}

			assert.ErrorIs(t, writeAwsStreamEvent(run.upstream, awsClaudePingEvent), io.ErrClosedPipe, "upstream must be closed")
			require.Nil(t, result.err)
			require.NotNil(t, result.usage)
			assert.Equal(t, 100, result.usage.PromptTokens)

			status := run.info.StreamStatus
			require.NotNil(t, status)
			assert.Equal(t, test.wantEndReason, status.EndReason)
			assert.Equal(t, test.disconnect, status.ClientGone)
			var errorMessages []string
			for _, entry := range status.Errors {
				errorMessages = append(errorMessages, entry.Message)
			}
			assert.Equal(t, test.wantErrors, errorMessages)
		})
	}
}

// A channel's same-channel retry matches the HTTP status Bedrock answered
// with, which only an SDK error carrying an HTTP response reports.
func TestAwsHandlersRecordUpstreamStatus(t *testing.T) {
	originalRelayTimeout := common.RelayTimeout
	common.RelayTimeout = 0
	t.Cleanup(func() {
		common.RelayTimeout = originalRelayTimeout
	})

	handlers := []struct {
		name    string
		request func() any
		handle  func(*gin.Context, *relaycommon.RelayInfo, *Adaptor) (*relaytypes.NewAPIError, *dto.Usage)
	}{
		{name: "non-stream", request: func() any { return newAwsInvokeModelInput() }, handle: awsHandler},
		{name: "stream", request: func() any { return newAwsStreamInput() }, handle: awsStreamHandler},
		{name: "nova", request: func() any { return newAwsInvokeModelInput() }, handle: handleNovaRequest},
	}
	upstreams := []struct {
		name               string
		do                 func(*http.Request) (*http.Response, error)
		wantStatus         int
		wantUpstreamStatus int
	}{
		{
			name: "throttled by Bedrock",
			do: func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Status:     "429 Too Many Requests",
					Header: http.Header{
						"Content-Type":      []string{"application/json"},
						"X-Amzn-Errortype":  []string{"ThrottlingException"},
						"X-Amzn-Requestid":  []string{"request-id"},
						"X-Amzn-Error-Code": []string{"ThrottlingException"},
					},
					Body:    io.NopCloser(bytes.NewReader([]byte(`{"message":"Too many requests"}`))),
					Request: request,
				}, nil
			},
			wantStatus:         http.StatusTooManyRequests,
			wantUpstreamStatus: http.StatusTooManyRequests,
		},
		{
			name: "connection failure",
			do: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			},
			wantStatus:         http.StatusInternalServerError,
			wantUpstreamStatus: 0,
		},
	}

	for _, handler := range handlers {
		for _, upstream := range upstreams {
			t.Run(handler.name+"/"+upstream.name, func(t *testing.T) {
				adaptor := &Adaptor{AwsClient: newAwsTestClient(awsHTTPClientFunc(upstream.do)), AwsReq: handler.request()}
				c := newAwsTestContext(httptest.NewRecorder(), context.Background())

				err, usage := handler.handle(c, newAwsTestRelayInfo(), adaptor)

				require.NotNil(t, err)
				assert.Nil(t, usage)
				assert.Equal(t, upstream.wantStatus, err.StatusCode)
				assert.Equal(t, upstream.wantUpstreamStatus, err.UpstreamStatusCode())
				assert.False(t, relaytypes.IsSkipRetryError(err))
			})
		}
	}
}
