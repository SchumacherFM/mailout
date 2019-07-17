package mailout

import (
	"errors"
	"testing"
	"time"

	"github.com/caddyserver/caddy"
	"github.com/stretchr/testify/assert"
)

func TestSetupParse(t *testing.T) {

	tests := []struct {
		config     string
		expectErr  error
		expectConf func() *config
	}{
		{
			`mailout`,
			nil,
			func() *config {
				return newConfig()
			},
		},
		{
			`mailout {
				recipient_to@domain.email 	testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.pgpEmailKeys = []string{`recipient_to@domain.email`, `testdata/B06469EE_nopw.pub.asc`}
				return c
			},
		},
		{
			`mailout /karate`,
			nil,
			func() *config {
				c := newConfig()
				c.endpoint = "/karate"
				return c
			},
		},
		{
			`mailout /kungfu {
				recipient_to@domain.email 	testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.endpoint = "/kungfu"
				c.pgpEmailKeys = []string{`recipient_to@domain.email`, `testdata/B06469EE_nopw.pub.asc`}
				return c
			},
		},
		{
			`mailout {
				recipient_to@domain.email testdata/B06469EE_nopw.pub.asc
				to              recipient_to@domain.email
				cc              "recipient_cc1@domain.email, recipient_cc2@domain.email"
				bcc             "recipient_bcc1@domain.email, recipient_bcc2@domain.email"
				subject         "Email from {{.firstname}} {{.lastname}}"
				body            testdata/mail_tpl.html
				host            127.0.0.1
				port            25
				skip_tls_verify
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.pgpEmailKeys = []string{`recipient_to@domain.email`, `testdata/B06469EE_nopw.pub.asc`}
				c.to = []string{"recipient_to@domain.email"}
				c.cc = []string{"recipient_cc1@domain.email", "recipient_cc2@domain.email"}
				c.bcc = []string{"recipient_bcc1@domain.email", "recipient_bcc2@domain.email"}
				c.subject = `Email from {{.firstname}} {{.lastname}}`
				c.body = `testdata/mail_tpl.html`
				c.host = "127.0.0.1"
				c.portRaw = "25"
				c.skipTLSVerify = true
				return c
			},
		},
		{
			`mailout /testFrom {
				recipient_to@domain.email testdata/B06469EE_nopw.pub.asc
				to              recipient_to@domain.email
				from_email	opensource@maintainer.org
				from_name	"Open Source Maintainer"
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.pgpEmailKeys = []string{"recipient_to@domain.email", "testdata/B06469EE_nopw.pub.asc"}
				c.endpoint = "/testFrom"
				c.to = []string{"recipient_to@domain.email"}
				c.fromEmail = "opensource@maintainer.org"
				c.fromName = "Open Source Maintainer"
				return c
			},
		},
		// <NOT_IMPLEMENTED>
		// You cannot define two endpoints for one host. this feature needs a total refactoring.
		// therefore the /sales endpoint gets overwritten by the /repairs endpoint.
		{
			`mailout /sales {
				salesteam@domain.email testdata/B06469EE_nopw.pub.asc
				to              salesteam@domain.email
				subject         "Sales Email from {{.firstname}} {{.lastname}}"
				body            testdata/mail_tpl.html
				host            127.0.0.1
				port            25
			}
			mailout /repairs {
				repairteam@domain.email testdata/B06469EE_nopw.pub.asc
				to              repairteam@domain.email
				subject         "Repair Email from {{.firstname}} {{.lastname}}"
				body            testdata/mail_tpl.html
				host            127.0.0.1
				port            25
			}
			`,
			nil,
			func() *config {
				c := newConfig()
				c.endpoint = "/repairs"
				c.pgpEmailKeys = []string{"salesteam@domain.email", "testdata/B06469EE_nopw.pub.asc", "repairteam@domain.email", "testdata/B06469EE_nopw.pub.asc"}
				c.to = []string{"repairteam@domain.email"}
				c.subject = `Repair Email from {{.firstname}} {{.lastname}}`
				c.body = `testdata/mail_tpl.html`
				c.host = "127.0.0.1"
				c.portRaw = "25"
				return c
			},
		},
		// </NOT_IMPLEMENTED>

		{
			`mailout {
				to              recipient_to@domain.email
				cc              "recipient_cc1@domain.email, recipient_cc2@domain.email"
				bcc             "recipient_bcc1@domain.email, recipient_bcc2@domain.email"
				recipient_to@domain.email 	testdata/B06469EE_nopw.pub.asc
				recipient_cc1@domain.email	https://keybase.io/cyrill/key.asc
				recipient_cc2@domain.email	testdata/B06469EE_nopw.pub.asc
				recipient_bcc2@domain.email	testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.pgpEmailKeys = []string{
					"recipient_to@domain.email", "testdata/B06469EE_nopw.pub.asc",
					"recipient_cc1@domain.email", "https://keybase.io/cyrill/key.asc",
					"recipient_cc2@domain.email", "testdata/B06469EE_nopw.pub.asc",
					"recipient_bcc2@domain.email", "testdata/B06469EE_nopw.pub.asc",
				}
				c.to = []string{"recipient_to@domain.email"}
				c.cc = []string{"recipient_cc1@domain.email", "recipient_cc2@domain.email"}
				c.bcc = []string{"recipient_bcc1@domain.email", "recipient_bcc2@domain.email"}
				return c
			},
		},
		{
			`mailout {
				to              recipient_to@domain.email
				recipient_to@domain.email 	testdata/B06469EE_nopw.pub.asc
				recipient_cc1@domain.email
			}`,
			errors.New("Testfile:4 - Error during parsing: Wrong argument count or unexpected line ending after 'recipient_cc1@domain.email'"),
			func() *config {
				c := newConfig()
				c.pgpEmailKeys = []string{
					"recipient_to@domain.email", "testdata/B06469EE_nopw.pub.asc",
				}
				c.to = []string{"recipient_to@domain.email"}
				return c
			},
		},
		{
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
		{
			`mailout /sendmail {
				to	"reci@email.de,"
			}`,
			errors.New("[mailout] Incorrect Email address found in: \"reci@email.de,\""),
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
		{
			`mailout /sendmail {
				to
				cc
				bcc
			}`,
			errors.New("Testfile:2 - Error during parsing: Wrong argument count or unexpected line ending after 'to'"),
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
		{
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
		{
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
		{
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
		{
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
		{
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
		{
			`mailout {
				publickeyAttachmentFileName
			}`,
			errors.New("Testfile:2 - Error during parsing: Wrong argument count or unexpected line ending after 'publickeyAttachmentFileName'"),
			func() *config {
				c := newConfig()
				c.pgpAttachmentName = "encrypted.asc"
				return c
			},
		},
		{
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
		{
			`mailout {
				ratelimit_interval
				ratelimit_capacity 500
			}`,
			errors.New("Testfile:2 - Error during parsing: Wrong argument count or unexpected line ending after 'ratelimit_interval'"),
			func() *config {
				c := newConfig()
				return c
			},
		},
		{
			`mailout {
				ratelimit_interval 6h
				ratelimit_capacity
			}`,
			errors.New("Testfile:3 - Error during parsing: Wrong argument count or unexpected line ending after 'ratelimit_capacity'"),
			func() *config {
				c := newConfig()
				return c
			},
		},
		{
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
		{
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
		c := caddy.NewTestController("http", test.config)
		mc, err := parse(c)
		if test.expectErr != nil {
			assert.Nil(t, mc, "Index %d", i)
			assert.EqualError(t, err, test.expectErr.Error(), "Index %d with config:\n%s", i, test.config)
			continue
		}
		assert.NoError(t, err, "Index %d", i)
		assert.Exactly(t, test.expectConf(), mc, "Index %d", i)
	}
}
