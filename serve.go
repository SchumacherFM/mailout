package mailout

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SchumacherFM/mailout/bufpool"
	"github.com/juju/ratelimit"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

// StatusUnprocessableEntity gets returned whenever parsing of the form fails.
const StatusUnprocessableEntity = 422

// StatusEmpty returned by mailout middleware because the proper status gets
// written previously
const StatusEmpty = 0

const (
	headerContentType         = "Content-Type"
	headerApplicationJSONUTF8 = "application/json; charset=utf-8"
)

func newHandler(mc *config, mailPipe chan<- *http.Request) *handler {
	return &handler{
		rlBucket: ratelimit.NewBucket(mc.rateLimitInterval, mc.rateLimitCapacity),
		reqPipe:  mailPipe,
		config:   mc,
	}
}

type handler struct {
	// rlBucket rate limit bucket
	rlBucket *ratelimit.Bucket
	// reqPipe send request to somewhere else. can be nil for testing.
	reqPipe chan<- *http.Request
	config  *config
	Next    httpserver.Handler
}

// ServeHTTP serves a request
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.URL.Path != h.config.endpoint {
		return h.Next.ServeHTTP(w, r)
	}

	if r.Method != "POST" {
		return h.writeJSON(JSONError{
			Code:  http.StatusMethodNotAllowed,
			Error: http.StatusText(http.StatusMethodNotAllowed),
		}, w)
	}

	if _, ok := h.rlBucket.TakeMaxDuration(1, h.config.rateLimitInterval); !ok {
		return h.writeJSON(JSONError{
			Code:  http.StatusTooManyRequests,
			Error: http.StatusText(http.StatusTooManyRequests),
		}, w)
	}

	if err := r.ParseForm(); err != nil {
		return h.writeJSON(JSONError{
			Code:  http.StatusBadRequest,
			Error: err.Error(),
		}, w)
	}

	if e := r.PostFormValue("email"); !isValidEmail(e) {
		return h.writeJSON(JSONError{
			Code:  StatusUnprocessableEntity,
			Error: fmt.Sprintf("Invalid email address: %q", e),
		}, w)
	}

	if h.reqPipe != nil {
		h.reqPipe <- r // might block if the mail daemon is busy
	}
	return h.writeJSON(JSONError{Code: http.StatusOK}, w)
}

// JSONError defines how an REST JSON looks like.
// Code 200 and empty Error specifies a successful request
// Any other Code value s an error.
type JSONError struct {
	// Code represents the HTTP Status Code, a work around.
	Code int `json:"code,omitempty"`
	// Error the underlying error, if there is one.
	Error string `json:"error,omitempty"`
}

func (h *handler) writeJSON(je JSONError, w http.ResponseWriter) (int, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	w.Header().Set(headerContentType, headerApplicationJSONUTF8)

	// https://github.com/mholt/caddy/issues/637#issuecomment-189599332
	w.WriteHeader(je.Code)

	if err := json.NewEncoder(buf).Encode(je); err != nil {
		return http.StatusInternalServerError, err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return http.StatusInternalServerError, err
	}

	return StatusEmpty, nil
}
