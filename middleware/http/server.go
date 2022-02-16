package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/d7561985/tel"
	"github.com/d7561985/tel/monitoring/metrics"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Server struct {
	log *tel.Telemetry
}

func NewServer(t *tel.Telemetry) *Server {
	return &Server{log: t}
}

// HTTPServerMiddlewareAll represent all essential metrics
// Execution order:
//  * opentracing injection via nethttp.Middleware
//  * recovery + measure execution time + debug log via own HTTPServerMiddleware
//  * metrics via metrics.NewHTTPMiddlewareWithOption
func (s *Server) HTTPServerMiddlewareAll(m metrics.HTTPTracker) func(next http.Handler) http.Handler {
	tr := func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "HTTP",
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return operation + r.Method + r.URL.Path
			}),
			otelhttp.WithFilter(func(r *http.Request) bool {
				return !(r.Method == http.MethodGet && strings.HasPrefix(r.URL.RequestURI(), "/health"))
			}))
	}

	mw := s.HTTPServerMiddleware()
	mtr := m.NewHTTPMiddlewareWithOption()

	return func(next http.Handler) http.Handler {
		for _, cb := range []func(next http.Handler) http.Handler{tr, mw, mtr} {
			next = cb(next)
		}

		return next
	}
}

// HTTPServerMiddleware perform:
// * telemetry log injection
// * measure execution time
// * recovery
func (s *Server) HTTPServerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(rw http.ResponseWriter, req *http.Request) {
			var err error

			// inject log
			// Warning! Don't use telemetry further, only via r.Context()
			r := req.WithContext(s.log.WithContext(req.Context()))
			w := metrics.NewHTTPStatusResponseWriter(rw)
			ctx := r.Context()

			// set tracing identification to log
			tel.UpdateTraceFields(ctx)

			var reqBody []byte
			if r.Body != nil {
				reqBody, _ = ioutil.ReadAll(r.Body)
				r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) // Reset
			}

			defer func(start time.Time) {
				hasRecovery := recover()

				l := tel.FromCtx(ctx).With(
					tel.Duration("duration", time.Since(start)),
					tel.String("method", r.Method),
					tel.String("user-agent", r.UserAgent()),
					tel.Any("req_header", r.Header),
					tel.String("ip", r.RemoteAddr),
					tel.String("path", r.URL.RequestURI()),
					tel.String("status_code", http.StatusText(w.Status)),
					tel.String("request", string(reqBody)),
				)

				if w.Response != nil {
					l = l.With(tel.String("response", string(w.Response)))
				}

				lvl := zapcore.DebugLevel
				if err != nil {
					lvl = zapcore.ErrorLevel
					l = l.With(tel.Error(err))
				}

				if hasRecovery != nil {
					lvl = zapcore.ErrorLevel
					l = l.With(tel.Error(fmt.Errorf("recovery info: %+v", hasRecovery)))

					// allow jaeger mw send error tag
					w.WriteHeader(http.StatusInternalServerError)
					if s.log.IsDebug() {
						debug.PrintStack()
					}
				}

				l.Check(lvl, fmt.Sprintf("HTTP %s %s", r.Method, r.URL.RequestURI())).Write()
			}(time.Now())

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}