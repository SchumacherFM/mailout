package mailout

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/SchumacherFM/mailout/bufpool"
	"github.com/juju/ratelimit"
	"github.com/mholt/caddy/middleware"
)

// StatusUnprocessableEntity gets returned whenever parsing of the form fails.
const StatusUnprocessableEntity = 422

const (
	// HeaderXRateLimitLimit - The number of allowed requests in the current period
	HeaderXRateLimitLimit = "X-Rate-Limit-Limit"
	// HeaderXRateLimitRemaining - The number of remaining requests in the current period
	HeaderXRateLimitRemaining = "X-Rate-Limit-Remaining"
	// HeaderXRateLimitReset - The number of seconds left in the current period
	HeaderXRateLimitReset = "X-Rate-Limit-Reset"
)

const (
	HeaderContentType         = "Content-Type"
	HeaderApplicationJSONUTF8 = "application/json; charset=utf-8"
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
	Next    middleware.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.URL.Path != h.config.endpoint {
		return h.Next.ServeHTTP(w, r)
	}

	var nextAvail time.Duration // always zero

	if r.Method != "POST" {
		return h.writeJSON(JSONError{
			Code:  http.StatusMethodNotAllowed,
			Error: http.StatusText(http.StatusMethodNotAllowed),
		}, w, nextAvail)
	}

	if td := h.rlBucket.Take(1); td > 0 {
		return h.writeJSON(JSONError{
			Code:  http.StatusTooManyRequests,
			Error: http.StatusText(http.StatusTooManyRequests),
		}, w, td)
	}

	if err := r.ParseForm(); err != nil {
		return h.writeJSON(JSONError{
			Code:  http.StatusBadRequest,
			Error: err.Error(),
		}, w, nextAvail)
	}

	if e := r.PostFormValue("email"); false == isValidEmail(e) {
		return h.writeJSON(JSONError{
			Code:  StatusUnprocessableEntity,
			Error: fmt.Sprintf("Invalid email address: %q", e),
		}, w, nextAvail)
	}

	if h.reqPipe != nil {
		h.reqPipe <- r
	}
	return h.writeJSON(JSONError{Code: http.StatusOK}, w, nextAvail)
}

type JSONError struct {
	// Code represents the HTTP Status Code, a work around.
	Code  int
	Error string `json:"error,omitempty"`
}

func (h *handler) writeJSON(je JSONError, w http.ResponseWriter, nextAvailable time.Duration) (int, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	w.Header().Set(HeaderContentType, HeaderApplicationJSONUTF8)
	w.Header().Set(HeaderXRateLimitLimit, strconv.FormatInt(h.rlBucket.Capacity()-h.rlBucket.Available(), 10))
	w.Header().Set(HeaderXRateLimitRemaining, strconv.FormatInt(h.rlBucket.Available(), 10))
	w.Header().Set(HeaderXRateLimitReset, strconv.FormatInt(int64(nextAvailable.Seconds()), 10))

	// https://github.com/mholt/caddy/wiki/Writing-Middleware#return-values-and-writing-responses
	// that does not play well with RESTful API design ....
	w.WriteHeader(je.Code) // caddy always prints out errors >= 400 codes and that breaks this API.

	if err := json.NewEncoder(buf).Encode(je); err != nil {
		return http.StatusInternalServerError, err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil // so caddy always gets a 200 from us
}
