package audit

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	mu     sync.Mutex
	events []Event
}

func (o *recordingObserver) Observe(event Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}

func TestSubjectNotifiesAllObservers(t *testing.T) {
	t.Parallel()

	first := &recordingObserver{}
	second := &recordingObserver{}
	subject := NewSubject()
	t.Cleanup(subject.Close)
	subject.Subscribe(first)
	subject.Subscribe(second)

	event := Event{TS: 123, Metrics: []string{"Alloc"}, IPAddress: "192.168.0.42"}
	subject.Notify(event)
	subject.Close()

	require.Len(t, first.events, 1)
	require.Len(t, second.events, 1)
	assert.Equal(t, event, first.events[0])
	assert.Equal(t, event, second.events[0])
}

func TestSubjectNilIsSafe(t *testing.T) {
	t.Parallel()

	var subject *Subject
	assert.NotPanics(t, func() {
		subject.Subscribe(&recordingObserver{})
		subject.Notify(Event{TS: 1})
		subject.NotifyRequest(httptest.NewRequest(http.MethodPost, "/", nil), []string{"Alloc"})
		subject.Close()
	})
}

func TestNotifyRequestSkipsEmptyMetrics(t *testing.T) {
	t.Parallel()

	rec := &recordingObserver{}
	subject := NewSubject()
	t.Cleanup(subject.Close)
	subject.Subscribe(rec)

	subject.NotifyRequest(httptest.NewRequest(http.MethodPost, "/", nil), nil)
	subject.Close()
	assert.Empty(t, rec.events)
}

type blockingObserver struct {
	started chan struct{}
	unblock chan struct{}
}

func (o blockingObserver) Observe(Event) {
	close(o.started)
	<-o.unblock
}

func TestNotifyDoesNotBlockOnSlowObserver(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	unblock := make(chan struct{})
	subject := NewSubject()
	t.Cleanup(func() {
		select {
		case <-unblock:
		default:
			close(unblock)
		}
		subject.Close()
	})
	subject.Subscribe(blockingObserver{started: started, unblock: unblock})

	done := make(chan struct{})
	go func() {
		subject.Notify(Event{TS: 1})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Notify blocked on slow observer")
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("observer was not called")
	}
	close(unblock)
}

func TestCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	subject := NewSubject()
	t.Cleanup(subject.Close)
	subject.Notify(Event{TS: 1})
	subject.Close()
	assert.NotPanics(t, subject.Close)
}

func TestFileObserverAppendsJSONLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "audit.log")
	observer, err := NewFileObserver(path, slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { _ = observer.Close() })

	observer.Observe(Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "10.0.0.1"})
	observer.Observe(Event{TS: 2, Metrics: []string{"Frees", "Mallocs"}, IPAddress: "10.0.0.2"})
	require.NoError(t, observer.Close())

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2)

	var first, second Event
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &second))
	assert.Equal(t, []string{"Alloc"}, first.Metrics)
	assert.Equal(t, []string{"Frees", "Mallocs"}, second.Metrics)
	assert.Equal(t, "10.0.0.2", second.IPAddress)
}

func TestNewFileObserverOpenError(t *testing.T) {
	t.Parallel()

	_, err := NewFileObserver(t.TempDir(), slog.Default())
	require.Error(t, err)
}

func TestFileObserverCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	observer, err := NewFileObserver(filepath.Join(t.TempDir(), "audit.log"), slog.Default())
	require.NoError(t, err)
	require.NoError(t, observer.Close())
	assert.NoError(t, observer.Close())
}

func TestHTTPObserverPostsJSON(t *testing.T) {
	t.Parallel()

	var got Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	NewHTTPObserver(server.URL, slog.Default()).Observe(Event{
		TS:        12345678,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	})

	assert.Equal(t, int64(12345678), got.TS)
	assert.Equal(t, []string{"Alloc", "Frees"}, got.Metrics)
	assert.Equal(t, "192.168.0.42", got.IPAddress)
}

func TestClientIPStripsPort(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	req.RemoteAddr = "192.168.0.42:54321"
	assert.Equal(t, "192.168.0.42", clientIP(req))
}
