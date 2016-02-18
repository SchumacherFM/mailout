package mailout

import (
	"net/http"

	"github.com/mholt/caddy/middleware"
)

type handler struct {
	Paths []string
	Next  middleware.Handler
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	// if the request path is any of the configured paths
	// write hello
	for _, p := range h.Paths {
		if middleware.Path(r.URL.Path).Matches(p) {
			w.Write([]byte("Hello, I'm a caddy middleware"))
			return 200, nil
		}
	}
	return h.Next.ServeHTTP(w, r)
}
