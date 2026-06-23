package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/alexey-y-a/bank-api/pkg/logger"
	"github.com/sirupsen/logrus"
)

type panicResponseWriter struct {
	http.ResponseWriter
	written bool
}

func newPanicResponseWriter(w http.ResponseWriter) *panicResponseWriter {
	return &panicResponseWriter{ResponseWriter: w}
}

func (pw *panicResponseWriter) WriteHeader(code int) {
	if !pw.written {
		pw.written = true
		pw.ResponseWriter.WriteHeader(code)
	}
}

func Recover(log *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pw := newPanicResponseWriter(w)

			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				requestID := GetRequestID(r.Context())

				stackTrace := string(debug.Stack())

				log.WithFields(logger.Fields{
					"request_id": requestID,
					"panic":      fmt.Sprintf("%v", rec),
					"stack":      stackTrace,
				}).Error("panic recovered in http handler")

				pw.WriteHeader(http.StatusInternalServerError)
				_, _ = pw.Write([]byte("internal server error"))
			}()

			next.ServeHTTP(pw, r)
		})
	}
}
