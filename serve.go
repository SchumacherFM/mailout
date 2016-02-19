package mailout

import (
	"net/http"

	"fmt"

	"github.com/mholt/caddy/middleware"
)

type handler struct {
	config *config
	Next   middleware.Handler
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	// if the request path is any of the configured paths
	// write hello

	if middleware.Path(r.URL.Path).Matches(h.config.endpoint) {
		fmt.Fprintf(w, "endpoint: %s", h.config.endpoint)
		w.Write([]byte("Hello, I'm a caddy middleware"))
		return 200, nil
	}

	return h.Next.ServeHTTP(w, r)
}
