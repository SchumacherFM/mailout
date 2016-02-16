package mailout

import (
	"fmt"
	"net/http"

	"github.com/mholt/caddy/caddy/setup"
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

func Setup(c *setup.Controller) (middleware.Middleware, error) {
	paths, err := parse(c)
	if err != nil {
		return nil, err
	}

	// Runs on Caddy startup, useful for services or other setups.
	c.Startup = append(c.Startup, func() error {

		// start mail daemon
		// start "pgp daemon"

		fmt.Println("mailout middleware is initiated")
		return nil
	})

	// Runs on Caddy shutdown, useful for cleanups.
	c.Shutdown = append(c.Shutdown, func() error {
		// quit mail daemon
		fmt.Println("mailout middleware is cleaning up")
		return nil
	})

	return func(next middleware.Handler) middleware.Handler {
		return &handler{
			Paths: paths,
			Next:  next,
		}
	}, nil
}

func parse(c *setup.Controller) ([]string, error) {
	// This parses the following config blocks
	/*
		mailout /hello
		mailout /anotherpath
		mailout {
			path /hello
			path /anotherpath
		}
	*/
	var paths []string
	for c.Next() {
		args := c.RemainingArgs()
		switch len(args) {
		case 0:
			// no argument passed, check the config block
			for c.NextBlock() {
				switch c.Val() {
				case "path":
					if !c.NextArg() {
						// we are expecting a value
						return paths, c.ArgErr()
					}
					p := c.Val()
					paths = append(paths, p)
					if c.NextArg() {
						// we are expecting only one value.
						return paths, c.ArgErr()
					}
				}
			}
		case 1:
			// one argument passed
			paths = append(paths, args[0])
			if c.NextBlock() {
				// path specified, no block required.
				return paths, c.ArgErr()
			}
		default:
			// we want only one argument max
			return paths, c.ArgErr()
		}
	}
	return paths, nil
}
