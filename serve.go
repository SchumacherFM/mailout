package mailout

import (
	"net/http"

	"fmt"

	"encoding/json"

	"github.com/SchumacherFM/mailout/bufpool"
	"github.com/mholt/caddy/middleware"
)

const StatusUnprocessableEntity = 422

type handler struct {
	reqPipe chan<- *http.Request
	config  *config
	Next    middleware.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.URL.Path != h.config.endpoint {
		return h.Next.ServeHTTP(w, r)
	}
	if r.Method != "POST" {
		return http.StatusMethodNotAllowed, nil
	}

	// TODO: juju/ratelimit e.g. 1000 mails per day or within 24h
	if false {
		//Add headers:
		//X-Rate-Limit-Limit - The number of allowed requests in the current period
		//X-Rate-Limit-Remaining - The number of remaining requests in the current period
		//X-Rate-Limit-Reset - The number of seconds left in the current period
		return http.StatusTooManyRequests, nil
	}

	if err := r.ParseForm(); err != nil {
		return http.StatusBadRequest, err
	}

	if e := r.PostFormValue("email"); false == isValidEmail(e) {
		return writeJSON(JSONError{
			Error: fmt.Sprintf("Invalid email address: %q", e),
		}, StatusUnprocessableEntity, w)
	}

	h.reqPipe <- r

	return writeJSON(JSONError{}, http.StatusOK, w)
}

type JSONError struct {
	Error string `json:"error,omitempty"`
}

func writeJSON(je JSONError, code int, w http.ResponseWriter) (int, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(buf).Encode(je); err != nil {
		return http.StatusInternalServerError, err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return http.StatusInternalServerError, err
	}
	return code, nil
}
