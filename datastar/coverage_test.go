package datastar_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-sse"
	"github.com/stretchr/testify/require"
)

type fakeTemplComponent struct {
	html string
	err  error
}

func (f fakeTemplComponent) Render(_ context.Context, w io.Writer) error {
	if f.err != nil {
		return f.err
	}

	_, err := io.WriteString(w, f.html)
	return err
}

func TestNewBroadcasterWithBufferSize(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithBufferSize(128)
	require.NotNil(t, b)
	require.Equal(t, 0, b.SubscriberCount())
}

func TestBroadcasterOnUnsubscribeCallback(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcaster()

	var count int
	var mu sync.Mutex

	b.OnUnsubscribe(func() {
		mu.Lock()
		count++
		mu.Unlock()
	})

	disconnect := connectSubscriber(t, b)
	disconnect()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return count == 1
	}, 2*time.Second, 5*time.Millisecond)
}

func TestLastEventID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Last-Event-ID", "42")

	id := ds.LastEventID(req)
	require.False(t, id.IsZero())
}

func TestElementsTemplPatch(t *testing.T) {
	t.Parallel()

	patch, err := ds.ElementsTemplPatch(
		fakeTemplComponent{html: "<div>from templ</div>"},
		ds.WithSelectorID("main"),
	)
	require.NoError(t, err)

	evt := patch.Event()
	require.Equal(t, "datastar-patch-elements", evt.Event)
	require.Contains(t, evt.Data, "elements <div>from templ</div>")
}

func TestElementsTemplPatchError(t *testing.T) {
	t.Parallel()

	_, err := ds.ElementsTemplPatch(fakeTemplComponent{err: io.ErrUnexpectedEOF})
	require.Error(t, err)
}

func TestReadSignals(t *testing.T) {
	t.Parallel()

	body := strings.NewReader(`{"count": 5}`)
	req := httptest.NewRequest(http.MethodPost, "/events", body)

	var signals struct {
		Count int `json:"count"`
	}
	err := ds.ReadSignals(req, &signals)
	require.NoError(t, err)
	require.Equal(t, 5, signals.Count)
}

func TestScriptHandler(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(ds.ScriptHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/javascript; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestScriptTag(t *testing.T) {
	t.Parallel()

	tag := ds.ScriptTag("/datastar.js")
	require.Contains(t, tag, "<script")
	require.Contains(t, tag, "/datastar.js")
}

func TestVersion(t *testing.T) {
	t.Parallel()

	v := ds.Version()
	require.NotEmpty(t, v)
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	stream := sse.NewStream(w, httptest.NewRequest(http.MethodGet, "/", nil))

	err := ds.ErrorResponse(stream, "something broke", "ERR_500")
	require.NoError(t, err)

	body := w.Body.String()
	require.Contains(t, body, "datastar-patch-signals")
	require.Contains(t, body, "something broke")
}

func TestBroadcastWithReplayStore(t *testing.T) {
	t.Parallel()

	b := ds.NewBroadcasterWithReplay(10)
	disconnect := connectSubscriber(t, b)

	patch := ds.ElementsPatch("<div>stored</div>")
	b.Broadcast(patch)

	disconnect()
	require.Equal(t, 0, b.SubscriberCount())
}
