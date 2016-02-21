package mailout

import (
	"net/http"

	"fmt"

	"encoding/json"

	"github.com/SchumacherFM/mailout/bufpool"
	"github.com/mholt/caddy/middleware"
)

type handler struct {
	reqPipe chan<- *http.Request
	config  *config
	Next    middleware.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.URL.Path != h.config.endpoint || r.Method != "POST" {
		return h.Next.ServeHTTP(w, r)
	}

	if err := r.ParseForm(); err != nil {
		return http.StatusInternalServerError, err
	}

	if e := r.PostFormValue("email"); false == isValidEmail(e) {
		return writeJSON(JSONError{
			Error: fmt.Sprintf("Invalid email address: %q", e),
		}, w)
	}

	h.reqPipe <- r

	return writeJSON(JSONError{}, w)
}

type JSONError struct {
	Error string `json:"error,omitempty"`
}

func writeJSON(je JSONError, w http.ResponseWriter) (int, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(buf).Encode(je); err != nil {
		return http.StatusInternalServerError, err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
