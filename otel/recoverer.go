package otel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"

	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/platform-smith-labs/japi-core/v4/core"
	"github.com/platform-smith-labs/japi-core/v4/internal/spanstatus"
)

// ErrHandlerPanic replaces the value of a panic that Middleware re-raises
// (when no Recoverer runs inside the span), so the panic value never reaches
// net/http's own panic log.
var ErrHandlerPanic = errors.New("otel: handler panicked (value withheld)")

// PanicStatusDescription is the server span status description after a
// recovered panic.
const PanicStatusDescription = "panic"

// recordPanic records a panic on the request's spans without its value: an
// exception event with exception.type (the Go type of the value) and
// exception.stacktrace, and the server span pinned to status Error
// "panic" with error.type internal_error.
func recordPanic(ctx context.Context, p any, stack []byte) {
	trace.SpanFromContext(ctx).AddEvent("exception", trace.WithAttributes(
		attribute.String("exception.type", fmt.Sprintf("%T", p)),
		attribute.String("exception.stacktrace", string(stack)),
	))
	spanstatus.Pin(ctx, PanicStatusDescription, string(core.CodeInternalError))
}

// Recoverer recovers a panic in the handler and writes the generic 500
// internal_error Problem Details body (with trace_id). It records the panic
// on the span (exception.type and exception.stacktrace, status Error
// "panic") and logs one ERROR line through slog.Default with the request
// context, carrying the type and stack trace.
//
// The panic value itself is recorded nowhere: it can carry request data. A
// stack trace lists functions and raw argument words, not values.
//
// http.ErrAbortHandler is re-raised, as net/http expects. When the response
// has already started, or the connection was hijacked, nothing more is
// written. The router installs Recoverer inside the tracing middleware.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var started, hijacked bool
		ww := httpsnoop.Wrap(w, httpsnoop.Hooks{
			WriteHeader: func(f httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return func(code int) {
					if code >= 200 {
						started = true
					}
					f(code)
				}
			},
			Write: func(f httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return func(b []byte) (int, error) {
					started = true
					return f(b)
				}
			},
			ReadFrom: func(f httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
				return func(src io.Reader) (int64, error) {
					started = true
					return f(src)
				}
			},
			Flush: func(f httpsnoop.FlushFunc) httpsnoop.FlushFunc {
				return func() {
					started = true
					f()
				}
			},
			Hijack: func(f httpsnoop.HijackFunc) httpsnoop.HijackFunc {
				return func() (net.Conn, *bufio.ReadWriter, error) {
					hijacked = true
					return f()
				}
			},
		})

		defer func() {
			p := recover()
			if p == nil {
				return
			}
			if p == http.ErrAbortHandler {
				panic(p)
			}
			if cv, ok := p.(core.ContractViolation); ok {
				// Only raised under `go test`: let the test fail loudly.
				panic(cv)
			}
			ctx := r.Context()
			stack := debug.Stack()
			recordPanic(ctx, p, stack)
			slog.Default().LogAttrs(ctx, slog.LevelError, "panic recovered",
				slog.String("exception.type", fmt.Sprintf("%T", p)),
				slog.String("exception.stacktrace", string(stack)),
				slog.String("http.request.method", r.Method),
				slog.String("http.route", routePattern(r)),
			)
			if started || hijacked {
				return
			}
			_ = core.WriteProblem(w, r, core.Internal(nil), core.AlreadyRecorded())
		}()

		next.ServeHTTP(ww, r)
	})
}
