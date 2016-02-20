package mailout

import (
	"errors"
	"strings"

	"github.com/SchumacherFM/mailout/maillog"
	"github.com/mholt/caddy/caddy/setup"
	"github.com/mholt/caddy/middleware"
)

func Setup(c *setup.Controller) (mw middleware.Middleware, err error) {
	var mc *config
	mc, err = parse(c)
	if err != nil {
		return nil, err
	}

	if c.ServerBlockHostIndex == 0 {
		// only run when the first hostname has been loaded.
		if _, err = mc.maillog.Init(c.ServerBlockHosts...); err != nil {
			return
		}
		if err = mc.loadFromEnv(); err != nil {
			return
		}
		if err = mc.loadPGPKey(); err != nil {
			return
		}
		if err = mc.loadTemplate(); err != nil {
			return
		}
		if err = mc.pingSMTP(); err != nil {
			return
		}

		mailPipe := startMailDaemon(mc)

		c.ServerBlockStorage = &handler{
			reqPipe: mailPipe,
			config:  mc,
		}
	}

	c.Shutdown = append(c.Shutdown, func() error {
		if moh, ok := c.ServerBlockStorage.(*handler); ok {
			if moh.reqPipe != nil {
				close(moh.reqPipe)
				moh.reqPipe = nil
			}
		}
		return nil
	})

	if moh, ok := c.ServerBlockStorage.(*handler); ok { // moh = mailOutHandler ;-)
		mw = func(next middleware.Handler) middleware.Handler {
			moh.Next = next
			return moh
		}
		return
	}
	err = errors.New("mailout: Could not create the middleware")
	return
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
				mc.maillog = maillog.New(strings.TrimSpace(c.Val()))
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
