// Package app for activation backend app
package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Response interface
type Response interface {
	Status() int
	Err() error

	// header getter
	Header() http.Header
	// header setter
	WithHeader(k, v string) Response
}

// Handler interface
type Handler func(r *http.Request, w http.ResponseWriter) (interface{}, Response)

// WrapFunc is a helper wrapper to make implementing handlers easier
func wrapFunc(a Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
		}()

		object, result := a(r, w)

		w.Header().Set("Content-Type", "application/json")

		if result == nil {
			w.WriteHeader(http.StatusOK)
		} else {

			h := result.Header()
			for k := range h {
				for _, v := range h.Values(k) {
					w.Header().Add(k, v)
				}
			}

			w.WriteHeader(result.Status())
			if err := result.Err(); err != nil {
				object = struct {
					Error string `json:"err"`
				}{
					Error: err.Error(),
				}
			}
		}

		if err := json.NewEncoder(w).Encode(object); err != nil {
			log.Error().Err(err).Msg("failed to encode return object")
		}
	}
}

type genericResponse struct {
	status int
	err    error
	header http.Header
}

func (r genericResponse) Status() int {
	return r.status
}

func (r genericResponse) Err() error {
	return r.err
}

func (r genericResponse) Header() http.Header {
	if r.header == nil {
		r.header = http.Header{}
	}
	return r.header
}

func (r genericResponse) WithHeader(k, v string) Response {
	if r.header == nil {
		r.header = http.Header{}
	}

	r.header.Add(k, v)
	return r
}

// Ok return a ok response
func ok() Response {
	return genericResponse{status: http.StatusOK}
}

// genError generic error response
func genError(err error, code int) Response {
	if err == nil {
		err = fmt.Errorf("no message")
	}

	return genericResponse{status: code, err: err}
}

// BadRequest result
func badRequest(err error) Response {
	return genError(err, http.StatusBadRequest)
}

// InternalServerError result
func internalServerError(err error) Response {
	return genError(err, http.StatusInternalServerError)
}

// NotFound response
func notFound(err error) Response {
	return genError(err, http.StatusNotFound)
}
