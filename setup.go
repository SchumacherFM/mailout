package mailout

import (
	"fmt"
	"strings"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/mholt/caddy/middleware"
)

const emailSplitBy = ","

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
	mc = newConfig()

	for c.Next() {
		args := c.RemainingArgs()

		switch len(args) {
		case 1:
			mc.endpoint = args[0]
		}

		for c.NextBlock() {
			var err error
			switch c.Val() {
			case "public_key":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.publicKey = c.Val()
			case "logdir":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}

				mc.maillog = newMailLogger(c.Val())

			case "success_uri":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.successUri = c.Val()
			case "to":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.to, err = splitEmailAddresses(c.Val())
				if err != nil {
					return nil, err
				}
			case "cc":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.cc, err = splitEmailAddresses(c.Val())
				if err != nil {
					return nil, err
				}
			case "bcc":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.bcc, err = splitEmailAddresses(c.Val())
				if err != nil {
					return nil, err
				}
			case "subject":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.subject = c.Val()
			case "body":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.body = c.Val()
			case "username":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.username = c.Val()
			case "password":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.password = c.Val()
			case "host":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.host = c.Val()
			case "port":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.portRaw = c.Val()
			}
		}
	}

	return
}

func splitEmailAddresses(s string) ([]string, error) {
	// maybe we're adding validation
	ret := strings.Split(s, emailSplitBy)
	for i, val := range ret {
		ret[i] = strings.TrimSpace(val)
		if ret[i] == "" {
			return nil, fmt.Errorf("Empty Email address found in: %q", s)
		}
	}
	return ret, nil
}
