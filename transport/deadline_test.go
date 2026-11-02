package transport

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

// deadlineSpy records every SetWriteDeadline call http.ResponseController makes
// on it. It implements the deadline interface directly, which is how
// *http.response exposes it in production.
type deadlineSpy struct {
	*httptest.ResponseRecorder

	calls   int
	cleared bool
	last    time.Time
}

func newDeadlineSpy() *deadlineSpy {
	return &deadlineSpy{
		ResponseRecorder: httptest.NewRecorder(),
		calls:            0,
		cleared:          false,
		last:             time.Time{},
	}
}

func (s *deadlineSpy) SetWriteDeadline(t time.Time) error {
	s.calls++
	s.last = t

	if t.IsZero() {
		s.cleared = true
	}

	return nil
}

// deadlineSpyWrapper hides the deadline interface behind an Unwrap hop, the
// shape every middleware that wraps the writer produces. Clearing must still
// reach the underlying connection.
type deadlineSpyWrapper struct {
	inner *deadlineSpy
}

func (w deadlineSpyWrapper) Header() http.Header         { return w.inner.Header() }
func (w deadlineSpyWrapper) Write(b []byte) (int, error) { return w.inner.Write(b) }
func (w deadlineSpyWrapper) WriteHeader(code int)        { w.inner.WriteHeader(code) }
func (w deadlineSpyWrapper) Unwrap() http.ResponseWriter { return w.inner }

func TestClearWriteDeadline_IgnoresUnsupportedWriter(t *testing.T) {
	t.Parallel()

	// httptest.ResponseRecorder has no deadline carrier. The helper must
	// swallow ErrNotSupported rather than panic or surface an error.
	ClearWriteDeadline(httptest.NewRecorder())
}

func TestClearWriteDeadline_ReachesUnderlyingWriter(t *testing.T) {
	t.Parallel()

	spy := newDeadlineSpy()

	ClearWriteDeadline(deadlineSpyWrapper{inner: spy})

	if !spy.cleared {
		t.Fatalf("expected a zero-time deadline to reach the wrapped writer, got %d call(s)", spy.calls)
	}
}

func TestServeDomainEvents_ClearsWriteDeadline(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	h := ServeDomainEvents(b, nil, 0)

	ctx, cancel := context.WithCancel(context.Background())

	spy := newDeadlineSpy()
	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	done := make(chan struct{})

	go func() {
		h.ServeHTTP(spy, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if !spy.cleared {
		t.Fatal("ServeDomainEvents must clear the connection write deadline")
	}
}

func TestServeDomainEvents_UnavailableKeepsWriteDeadline(t *testing.T) {
	t.Parallel()

	// The 503 path never opens a stream, so it must not relax the deadline:
	// a caller that misconfigures the feed should not get a sticky
	// no-timeout connection out of an error response.
	h := ServeDomainEvents(nil, nil, 0)

	spy := newDeadlineSpy()
	h.ServeHTTP(spy, httptest.NewRequest(http.MethodGet, "/events", nil))

	if spy.calls != 0 {
		t.Fatalf("503 path should not touch the write deadline, got %d call(s)", spy.calls)
	}
}

// TestSSESurvivesServerWriteTimeout is the regression this fix exists for: a
// server with a short WriteTimeout used to sever every SSE stream that stayed
// open longer than the timeout. It also pins the other half of the contract —
// a NON-streaming route on the same server keeps its deadline, so the fix
// relaxes one connection, not the server.
func TestSSESurvivesServerWriteTimeout(t *testing.T) {
	const (
		writeTimeout = 50 * time.Millisecond
		deliverAfter = 200 * time.Millisecond
	)

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	mux := http.NewServeMux()
	mux.Handle("GET /events", ServeDomainEvents(b, nil, 0))
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, _ *http.Request) {
		// Writes well past the server write timeout.
		time.Sleep(deliverAfter)

		_, _ = w.Write([]byte("late"))
	})

	srv := httptest.NewUnstartedServer(mux)
	srv.Config.WriteTimeout = writeTimeout

	srv.Start()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	go func() {
		time.Sleep(deliverAfter)
		b.Broadcast(sse.Event{Event: "ping", Data: "late-delivery"})
	}()

	scanner := bufio.NewScanner(resp.Body)

	delivered := false

	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "late-delivery") {
			delivered = true

			break
		}
	}

	if !delivered {
		t.Fatalf(
			"event sent after %s was not delivered through a %s WriteTimeout (scan err: %v)",
			deliverAfter, writeTimeout, scanner.Err(),
		)
	}

	// Same server, non-streaming route: the deadline must still apply.
	slowReq, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/slow", nil)
	if err != nil {
		t.Fatalf("build slow request: %v", err)
	}

	slowResp, err := srv.Client().Do(slowReq)
	if err != nil {
		// The connection was cut by the deadline — exactly the expected
		// behaviour for a non-SSE route.
		return
	}
	defer func() { _ = slowResp.Body.Close() }()

	body, _ := io.ReadAll(slowResp.Body)
	if string(body) == "late" {
		t.Fatal("non-SSE route wrote past WriteTimeout: the deadline was relaxed for the whole server")
	}
}
