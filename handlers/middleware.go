package handlers

import (
	"net/http"

	"github.com/cloudfoundry/dropsonde"
	"github.com/pivotal-golang/lager"
)

func LogWrap(handler http.Handler, logger lager.Logger) http.HandlerFunc {
	handler = dropsonde.InstrumentedHandler(handler)

	return func(w http.ResponseWriter, r *http.Request) {
		requestLog := logger.Session("request", lager.Data{
			"method":  r.Method,
			"request": r.URL.String(),
		})

		requestLog.Info("serving", lager.Data{"request-headers": filter(r.Header)})
		handler.ServeHTTP(w, r)
		requestLog.Info("done", lager.Data{"response-headers": w.Header()})
	}
}

func filter(header http.Header) http.Header {
	filtered := make(http.Header)
	for k, v := range header {
		// TODO: cleanup, case-insensitive, blacklist?
		if k != "Authorization" {
			filtered[k] = v
		}
	}
	return filtered
}
