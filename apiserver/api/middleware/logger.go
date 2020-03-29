package middleware

import (
	"context"
	"net/http"
	"time"

	chi_middleware "github.com/go-chi/chi/middleware"
	log "github.com/sirupsen/logrus"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "chi/middleware context value " + k.name
}

// LogEntryCtxKey is the context.Context key to store the request log entry.
var LogEntryCtxKey = &contextKey{"LogEntry"}

// RequestLogger returns a logger handler using a custom LogFormatter.
func RequestLogger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			requestStartTime := time.Now()

			fields := requestLogFields(r)

			ww := chi_middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				dur := time.Since(requestStartTime)
				fields["status"] = ww.Status()
				fields["bytes_written"] = ww.BytesWritten()
				fields["request_duration_seconds"] = dur.Seconds()
				log.WithFields(fields).Printf("%s %s %d %s", r.Method, r.RequestURI, ww.Status(), dur)
			}()

			next.ServeHTTP(ww, withLogFields(r, &fields))
		}
		return http.HandlerFunc(fn)
	}
}

// RequestLogFields returns the in-context LogEntry for a request.
func RequestLogFields(r *http.Request) log.Fields {
	v := r.Context().Value(LogEntryCtxKey)
	switch x := v.(type) {
	case log.Fields:
		return x
	default:
		return log.Fields{}
	}
}

// withLogFields sets the in-context LogEntry for a request.
func withLogFields(r *http.Request, fields *log.Fields) *http.Request {
	r = r.WithContext(context.WithValue(r.Context(), LogEntryCtxKey, fields))
	return r
}

// requestLogFields creates a new LogEntry for the request.
func requestLogFields(r *http.Request) log.Fields {
	return log.Fields{
		"method":         r.Method,
		"request_uri":    r.RequestURI,
		"remote_address": r.RemoteAddr,
	}
}
