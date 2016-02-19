package mailout

import (
	"errors"
	"testing"

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
		{
			`mailout`,
			nil,
			func() *config {
				return newConfig()
			},
		},
		{
			`mailout {
				public_key testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.publicKey = `testdata/B06469EE_nopw.pub.asc`
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
				public_key testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			func() *config {
				c := newConfig()
				c.endpoint = "/kungfu"
				c.publicKey = `testdata/B06469EE_nopw.pub.asc`
				return c
			},
		},
		{
			`mailout {
				public_key testdata/B06469EE_nopw.pub.asc
				success_uri     email_sent_confirmation.html
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
			errors.New("Incorrect Email address found in: \"reci@email.de,\""),
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
				to
				cc
				bcc
			}`,
			errors.New("Testfile:2 - Parse error: Wrong argument count or unexpected line ending after 'to'"),
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
