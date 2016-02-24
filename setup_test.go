package mailout

import (
	"errors"
	"testing"

	"time"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/stretchr/testify/assert"
)

func TestSetupParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		config     string
		expectErr  error
		expectConf func() *config
	}{
		0: {
			`mailout`,
			nil,
			func() *config {
				return newConfig()
			},
		},
		1: {
			`mailout {
				publickey testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.publicKey = `testdata/B06469EE_nopw.pub.asc`
				return c
			},
		},
		2: {
			`mailout /karate`,
			nil,
			func() *config {
				c := newConfig()
				c.endpoint = "/karate"
				return c
			},
		},
		3: {
			`mailout /kungfu {
				publickey testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.endpoint = "/kungfu"
				c.publicKey = `testdata/B06469EE_nopw.pub.asc`
				return c
			},
		},
		4: {
			`mailout {
				publickey testdata/B06469EE_nopw.pub.asc
				to              recipient_to@domain.email
				cc              "recipient_cc1@domain.email, recipient_cc2@domain.email"
				bcc             "recipient_bcc1@domain.email, recipient_bcc2@domain.email"
				subject         "Email from {{.firstname}} {{.lastname}}"
				body            testdata/mail_tpl.html
				host            127.0.0.1
				port            25
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.publicKey = `testdata/B06469EE_nopw.pub.asc`
				c.to = []string{"recipient_to@domain.email"}
				c.cc = []string{"recipient_cc1@domain.email", "recipient_cc2@domain.email"}
				c.bcc = []string{"recipient_bcc1@domain.email", "recipient_bcc2@domain.email"}
				c.subject = `Email from {{.firstname}} {{.lastname}}`
				c.body = `testdata/mail_tpl.html`
				c.host = "127.0.0.1"
				c.portRaw = "25"
				return c
			},
		},
		5: {
			`mailout /sendmail {
				username 	g0ph3r
				password 	release1.6
				host        127.0.0.2
				port        25
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.endpoint = "/sendmail"
				c.username = "g0ph3r"
				c.password = "release1.6"
				c.host = "127.0.0.2"
				c.portRaw = "25"
				return c
			},
		},
		6: {
			`mailout /sendmail {
				to	"reci@email.de,"
			}`,
			errors.New("Incorrect Email address found in: \"reci@email.de,\""),
			func() *config {
				c := newConfig()
				c.endpoint = defaultEndpoint
				c.username = "g0ph3r"
				c.password = "release1.6"
				c.host = "127.0.0.2"
				c.portRaw = "25"
				return c
			},
		},
		7: {
			`mailout /sendmail {
				to
				cc
				bcc
			}`,
			errors.New("Testfile:2 - Parse error: Wrong argument count or unexpected line ending after 'to'"),
			func() *config {
				c := newConfig()
				c.endpoint = defaultEndpoint
				c.username = "g0ph3r"
				c.password = "release1.6"
				c.host = "127.0.0.2"
				c.portRaw = "25"
				return c
			},
		},
		8: {
			`mailout {
				maillog testdata
				errorlog testdata
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.maillog.MailDir = "testdata"
				c.maillog.ErrDir = "testdata"
				return c
			},
		},
		9: {
			`mailout {
				maillog testdata
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.maillog.MailDir = "testdata"
				return c
			},
		},
		10: {
			`mailout {
				errorlog testdata
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.maillog.ErrDir = "testdata"
				return c
			},
		},
		11: {
			`mailout {
				errorlog testdata
				maillog testdata
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.maillog.MailDir = "testdata"
				c.maillog.ErrDir = "testdata"
				return c
			},
		},
		12: {
			`mailout {
				publickeyAttachmentFileName "encrypted.asc"
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.pgpAttachmentName = "encrypted.asc"
				return c
			},
		},
		13: {
			`mailout {
				publickeyAttachmentFileName
			}`,
			errors.New("Testfile:2 - Parse error: Wrong argument count or unexpected line ending after 'publickeyAttachmentFileName'"),
			func() *config {
				c := newConfig()
				c.pgpAttachmentName = "encrypted.asc"
				return c
			},
		},
		14: {
			`mailout {
				ratelimit_interval 12h
				ratelimit_capacity 500
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.rateLimitInterval = time.Hour * 12
				c.rateLimitCapacity = 500
				return c
			},
		},
		15: {
			`mailout {
				ratelimit_interval
				ratelimit_capacity 500
			}`,
			errors.New("Testfile:2 - Parse error: Wrong argument count or unexpected line ending after 'ratelimit_interval'"),
			func() *config {
				c := newConfig()
				return c
			},
		},
		16: {
			`mailout {
				ratelimit_interval 6h
				ratelimit_capacity
			}`,
			errors.New("Testfile:3 - Parse error: Wrong argument count or unexpected line ending after 'ratelimit_capacity'"),
			func() *config {
				c := newConfig()
				return c
			},
		},
		17: {
			`mailout {
				ratelimit_interval 12x
				ratelimit_capacity 500
			}`,
			errors.New("time: unknown unit x in duration 12x"),
			func() *config {
				c := newConfig()
				c.rateLimitCapacity = 500
				return c
			},
		},
		18: {
			`mailout {
				ratelimit_interval 12s
				ratelimit_capacity 5x
			}`,
			errors.New("strconv.ParseInt: parsing \"5x\": invalid syntax"),
			func() *config {
				c := newConfig()
				c.rateLimitInterval = time.Second * 12
				return c
			},
		},
	}
	for i, test := range tests {

		c := setup.NewTestController(test.config)
		mc, err := parse(c)
		if test.expectErr != nil {
			assert.Nil(t, mc, "Index %d", i)
			assert.EqualError(t, err, test.expectErr.Error(), "Index %d", i)
			continue
		}
		assert.NoError(t, err, "Index %d", i)
		assert.Exactly(t, test.expectConf(), mc, "Index %d", i)
	}
}
