package core

import (
	"context"
	"errors"
	"net/http"
)

// HandlerFunc is an http handler that returns an error.
//
// A returned error is written with WriteError: a *Problem as constructed,
// anything else as the generic 500 internal_error problem (its text is logged,
// never sent). A context.Canceled error (the client went away) writes nothing.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP calls h and writes its error, if any.
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		_ = WriteError(w, r, err)
	}
}

// WrapHandler converts a regular http.HandlerFunc to our HandlerFunc
func WrapHandler(h http.HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		h(w, r)
		return nil
	}
}
