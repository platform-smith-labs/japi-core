package router_test

import (
	"bufio"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/platform-smith-labs/japi-core/v4/metrics"
	httpmiddleware "github.com/platform-smith-labs/japi-core/v4/middleware/http"
	"github.com/platform-smith-labs/japi-core/v4/otel"
	"github.com/platform-smith-labs/japi-core/v4/router"
)

const sseGap = 50 * time.Millisecond

var (
	chainOnce  sync.Once
	chainSrv   *httptest.Server
	chainSpans *tracetest.SpanRecorder
	// deadlineErr records SetWriteDeadline's result inside the WebSocket handler.
	deadlineErr = make(chan error, 1)
)

// fullChain builds the complete stack once per test binary (metrics registers
// on the default Prometheus registerer, which accepts a collector only once):
// RequestID -> RealIP -> tracing -> Recoverer -> CORS -> logging -> metrics -> handler.
func fullChain(t *testing.T) (*httptest.Server, *tracetest.SpanRecorder) {
	t.Helper()
	chainOnce.Do(func() {
		chainSpans = tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(chainSpans))

		r := router.NewChiRouterWithOptions(router.WithTracing("test", otel.WithTracerProvider(tp)))
		r.Use(httpmiddleware.WithLogging(slog.New(slog.NewTextHandler(io.Discard, nil))))
		metrics.EnablePrometheusMetrics(r, "/metrics")

		r.Get("/events", sseHandler)
		r.Get("/ws", wsHandler)
		chainSrv = httptest.NewServer(r)
	})
	return chainSrv, chainSpans
}

func sseHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	rc := http.NewResponseController(w)
	for i := range 3 {
		if i > 0 {
			time.Sleep(sseGap)
		}
		_, _ = io.WriteString(w, "data: "+string(rune('0'+i))+"\n\n")
		if err := rc.Flush(); err != nil {
			panic("flush through middleware chain failed: " + err.Error())
		}
	}
}

func wsHandler(w http.ResponseWriter, _ *http.Request) {
	rc := http.NewResponseController(w)
	deadlineErr <- rc.SetWriteDeadline(time.Now().Add(5 * time.Second))
	conn, brw, err := rc.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	_, _ = brw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n\r\n")
	_ = brw.Flush()
	// Echo one line to prove the hijacked connection is usable.
	line, _ := brw.ReadString('\n')
	_, _ = brw.WriteString(line)
	_ = brw.Flush()
}

// TestFullChain_SSEStreamsEachEventBeforeTheNext verifies every wrapper in the
// chain forwards Flush: the client receives each event while the handler is
// still sleeping before the next one.
func TestFullChain_SSEStreamsEachEventBeforeTheNext(t *testing.T) {
	srv, _ := fullChain(t)
	resp, err := http.Get(srv.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	start := time.Now()
	br := bufio.NewReader(resp.Body)
	var arrivals []time.Duration
	for i := range 3 {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("event %d: %v", i, err)
		}
		if want := "data: " + string(rune('0'+i)) + "\n"; line != want {
			t.Fatalf("event %d = %q, want %q", i, line, want)
		}
		_, _ = br.ReadString('\n') // blank separator
		arrivals = append(arrivals, time.Since(start))
	}
	// Event i is written at ~i*gap. Had it been buffered until the handler
	// returned, all three would arrive together at ~2*gap.
	if arrivals[0] >= sseGap {
		t.Errorf("first event arrived after %v, want before the first gap (%v): response buffered", arrivals[0], sseGap)
	}
	for i := 1; i < 3; i++ {
		if d := arrivals[i] - arrivals[i-1]; d < sseGap/2 {
			t.Errorf("event %d arrived %v after event %d, want ~%v: events were batched", i, d, i-1, sseGap)
		}
	}
}

// TestFullChain_WebSocketUpgrade verifies a hijack-based upgrade and
// SetWriteDeadline work through every wrapper, and the span ends at the 101.
func TestFullChain_WebSocketUpgrade(t *testing.T) {
	srv, spans := fullChain(t)
	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, _ = io.WriteString(conn, "GET /ws HTTP/1.1\r\nHost: example\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n"+
		"Sec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n")

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}
	if err := <-deadlineErr; err != nil {
		t.Errorf("SetWriteDeadline through the chain: %v", err)
	}

	_, _ = io.WriteString(conn, "ping\n")
	if echo, err := br.ReadString('\n'); err != nil || echo != "ping\n" {
		t.Fatalf("echo over hijacked conn = %q, %v", echo, err)
	}

	var found bool
	for _, s := range spans.Ended() {
		if s.Name() == "GET /ws" {
			found = true
		}
	}
	if !found {
		t.Error("no ended span named \"GET /ws\"")
	}
}

// TestWithTracing_OffByDefault verifies a router without WithTracing
// and serves normally.
func TestWithTracing_OffByDefault(t *testing.T) {
	r := router.NewChiRouter()
	r.Get("/x", func(w http.ResponseWriter, r *http.Request) {
		if chi.RouteContext(r.Context()) == nil {
			t.Error("missing route context")
		}
		_, _ = io.WriteString(w, "ok")
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("got %d %q", rec.Code, rec.Body.String())
	}
}
