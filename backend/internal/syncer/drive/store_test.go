package drive

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/api/googleapi"

	"cloud-saver/backend/internal/syncer"
)

func TestWrapDriveRateLimitCarriesRetryAfter(t *testing.T) {
	t.Parallel()
	err := &googleapi.Error{
		Code:    http.StatusTooManyRequests,
		Message: "Rate Limit Exceeded",
		Header:  http.Header{"Retry-After": []string{"120"}},
		Errors:  []googleapi.ErrorItem{{Reason: "userRateLimitExceeded"}},
	}
	wrapped := wrapDriveError(err)
	var providerErr *syncer.ProviderOperationError
	if !errors.As(wrapped, &providerErr) {
		t.Fatalf("expected provider operation error, got %T", wrapped)
	}
	if providerErr.Code != "rate_limited" || !providerErr.Retryable {
		t.Fatalf("unexpected classification: %#v", providerErr)
	}
	if providerErr.RetryAfter != 2*time.Minute {
		t.Fatalf("expected two-minute retry delay, got %s", providerErr.RetryAfter)
	}
}

func TestWrapDriveStorageQuotaDoesNotSpin(t *testing.T) {
	t.Parallel()
	err := &googleapi.Error{
		Code:    http.StatusForbidden,
		Message: "Storage quota exceeded",
		Errors:  []googleapi.ErrorItem{{Reason: "storageQuotaExceeded"}},
	}
	var providerErr *syncer.ProviderOperationError
	if !errors.As(wrapDriveError(err), &providerErr) {
		t.Fatal("expected provider operation error")
	}
	if providerErr.Code != "quota_full" || providerErr.Retryable {
		t.Fatalf("quota error must stop automatic retries: %#v", providerErr)
	}
}

func TestPacedTransportCancellationDoesNotReachNetwork(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	transport := &pacedTransport{base: base, interval: time.Hour}
	first, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.test/first", nil)
	response, err := transport.RoundTrip(first)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	second, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/second", nil)
	if _, err := transport.RoundTrip(second); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled paced request, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("canceled request reached transport; calls=%d", calls.Load())
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	value := now.Add(90 * time.Second).Format(http.TimeFormat)
	if got := parseRetryAfter(value, now); got != 90*time.Second {
		t.Fatalf("expected 90 seconds, got %s", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
