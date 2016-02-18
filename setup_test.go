package mailout

import (
	"testing"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/stretchr/testify/assert"
)

//const testConfig1 = `
//mailout /mySendMail {
//	public_key      testdata/B06469EE_nopw.pub.asc
//	logdir          testdata/
//
//	success_uri     email_sent_confirmation.html
//
//	to              recipient_to@domain.email
//	cc              recipient_cc1@domain.email, recipient_cc2@domain.email
//	bcc             recipient_bcc1@domain.email, recipient_bcc2@domain.email
//    subject         Email from {{.firstname}} {{.lastname}}
//	body            testdata/mail_tpl.html
//
//	username
//	password
//	host            127.0.0.1
//	port            1025
//}
//`

func TestSetupParse(t *testing.T) {
	tests := []struct {
		config     string
		expectErr  error
		expectConf *config
	}{
		{
			`mailout`,
			nil,
			&config{
			// todo
			},
		},
	}
	for i, test := range tests {

		c := setup.NewTestController(test.config)
		mc, err := parse(c)
		if test.expectErr != nil {
			assert.Nil(t, mc)
			assert.EqualError(t, err, test.expectErr.Error(), "Index %d", i)
			continue
		}
		assert.NoError(t, err, "Index %d", i)
		assert.Exactly(t, test.expectConf, mc)
	}
}
