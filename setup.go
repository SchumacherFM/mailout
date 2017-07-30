package mailout

import (
	"errors"
	"strconv"
	"time"

	"github.com/SchumacherFM/mailout/maillog"
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

func init() {
	caddy.RegisterPlugin("mailout", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

// setup used internally by Caddy to set up this middleware
func setup(c *caddy.Controller) error {
	mc, err := parse(c)
	if err != nil {
		return err
	}

	if c.ServerBlockKeyIndex == 0 {
		// only run when the first hostname has been loaded.
		if mc.maillog, err = mc.maillog.Init(c.ServerBlockKeys...); err != nil {
			return err
		}
		if err = mc.loadFromEnv(); err != nil {
			return err
		}
		if err = mc.loadPGPKeys(); err != nil {
			return err
		}
		if err = mc.loadTemplate(); err != nil {
			return err
		}
		if err = mc.pingSMTP(); err != nil {
			return err
		}

		c.ServerBlockStorage = newHandler(mc, startMailDaemon(mc))
	}

	c.OnShutdown(func() error {
		if moh, ok := c.ServerBlockStorage.(*handler); ok {
			if moh.reqPipe != nil {
				close(moh.reqPipe)
				moh.reqPipe = nil
			}
		}
		return nil
	})

	if moh, ok := c.ServerBlockStorage.(*handler); ok { // moh = mailOutHandler ;-)
		httpserver.GetConfig(c).AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
			moh.Next = next
			return moh
		})
		return nil
	}
	return errors.New("[mailout] Could not create the middleware handler")
}

func parse(c *caddy.Controller) (mc *config, _ error) {
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
			case "publickeyAttachmentFileName":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.pgpAttachmentName = c.Val()
			case "maillog":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				if mc.maillog.IsNil() {
					mc.maillog = maillog.New(c.Val(), "")
				} else {
					mc.maillog.MailDir = c.Val()
				}
			case "errorlog":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				if mc.maillog.IsNil() {
					mc.maillog = maillog.New("", c.Val())
				} else {
					mc.maillog.ErrDir = c.Val()
				}
			case "from_email":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.fromEmail = c.Val()
			case "from_name":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				mc.fromName = c.Val()
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
			case "skip_tls_verify":
				mc.skipTLSVerify = true
			case "ratelimit_interval":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				var rli time.Duration
				rli, err = time.ParseDuration(c.Val())
				if err != nil {
					return nil, err
				}
				if rli.Nanoseconds() != 0 {
					mc.rateLimitInterval = rli
				}
			case "ratelimit_capacity":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				var rlc int64
				rlc, err = strconv.ParseInt(c.Val(), 10, 64)
				if err != nil {
					return nil, err
				}
				if rlc > 0 {
					mc.rateLimitCapacity = rlc
				}
			default:
				anyKey := c.Val()
				if isValidEmail(anyKey) {
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					pgpPublicKey := c.Val()
					mc.pgpEmailKeys = append(mc.pgpEmailKeys, anyKey, pgpPublicKey)
				}
			}
		}
	}
	return
}
