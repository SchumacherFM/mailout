package mailout

import (
	"fmt"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/mholt/caddy/middleware"
)

func Setup(c *setup.Controller) (middleware.Middleware, error) {
	mc, err := parse(c)
	if err != nil {
		return nil, err
	}

	// Runs on Caddy startup, useful for services or other setups.
	c.Startup = append(c.Startup, func() error {

		if err := mc.loadFromEnv(); err != nil {
			return err
		}
		if err := mc.loadPGPKey(); err != nil {
			return err
		}

		fmt.Println("\nmailout middleware is initiated")
		return nil
	})

	// Runs on Caddy shutdown, useful for cleanups.
	c.Shutdown = append(c.Shutdown, func() error {
		// quit mail daemon
		fmt.Println("\nmailout middleware is cleaning up")
		return nil
	})

	return func(next middleware.Handler) middleware.Handler {
		return &handler{
			Paths: nil,
			Next:  next,
		}
	}, nil
}

func parse(c *setup.Controller) (mc *config, err error) {
	// This parses the following config blocks
	/*
		mailout /hello
		mailout /anotherpath
		mailout {
			path /hello
			path /anotherpath
		}
	*/

	for c.Next() {
		args := c.RemainingArgs()
		switch len(args) {
		case 0:
			// no argument passed, check the config block
			for c.NextBlock() {
				println(c.Val())
				//switch c.Val() {
				//case "path":
				//	if !c.NextArg() {
				//		// we are expecting a value
				//		return mc, c.ArgErr()
				//	}
				//	println( c.Val())
				//	//paths = append(paths, p)
				//	if c.NextArg() {
				//		// we are expecting only one value.
				//		return mc, c.ArgErr()
				//	}
				//}
			}
		case 1:
			// one argument passed
			//paths = append(mc, args[0])
			println("1:", c.Val())
			if c.NextBlock() {
				// path specified, no block required.
				return mc, c.ArgErr()
			}
		default:
			println("default", c.Val())
			// we want only one argument max
			return mc, c.ArgErr()
		}
	}
	return
}
