package mailout

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/SchumacherFM/mailout/maillog"
	"github.com/mholt/caddy"
	"github.com/stretchr/testify/assert"
)

func testMessageServer(t *testing.T, caddyFile string, buf *bytes.Buffer, expectedMsgCount int) *httptest.Server {
	c := caddy.NewTestController("http", caddyFile)
	mc, err := parse(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := mc.loadTemplate(); err != nil {
		t.Fatal(err)
	}
	if err := mc.loadPGPKeys(); err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, expectedMsgCount, mc.messageCount, "\nCaddyFile %s\n", caddyFile)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		msg := newMessage(mc, r).build()
		if _, err := msg.WriteTo(buf); err != nil {
			t.Fatal(err)
		}
	}))
}

func testDoPost(t *testing.T, url string, data url.Values) *http.Response {
	hClient := &http.Client{}
	//hClient.Timeout = time.Millisecond
	resp, err := hClient.PostForm(url, data)
	if err != nil {
		t.Fatal(err)
	}
	if 200 != resp.StatusCode {
		t.Fatalf("Want StatusCode 200, Have %d", resp.StatusCode)
	}
	return resp
}

func TestMessagePlainTextAllFormFields(t *testing.T) {

	const caddyFile = `mailout {
				to              gopher@domain.email
				cc              "gopher1@domain.email, gopher2@domain.email"
				subject         "Email from {{ .Form.Get \"firstname\" }} {{.Form.Get \"lastname\"}}"
				body            testdata/mail_plainTextMessage.txt
			}`

	buf := new(bytes.Buffer)
	srv := testMessageServer(t, caddyFile, buf, 1)
	defer srv.Close()

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("name", "Ken Thompson")

	testDoPost(t, srv.URL, data)

	assert.Len(t, buf.String(), 424) // whenever you change the template, change also here
	assert.Contains(t, buf.String(), "Email ken@thompson.email")
	assert.Contains(t, buf.String(), `From: "Ken Thompson" <ken@thompson.email>`)
	assert.Contains(t, buf.String(), "Subject: Email from Ken Thompson")
	assert.Contains(t, buf.String(), "Cc: gopher1@domain.email, gopher2@domain.email")
	//t.Log(buf.String())
}

func TestMessagePlainTextWithOutFormInputName(t *testing.T) {

	const caddyFile = `mailout {
				to              gopher@domain.email
				subject         "Email from {{ .Form.Get \"firstname\" }} {{.Form.Get \"lastname\"}}"
				body            testdata/mail_plainTextMessage.txt
			}`

	buf := new(bytes.Buffer)
	srv := testMessageServer(t, caddyFile, buf, 1)
	defer srv.Close()

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")

	testDoPost(t, srv.URL, data)

	assert.Len(t, buf.String(), 359) // whenever you change the template, change also here
	assert.Contains(t, buf.String(), "Email ken@thompson.email")
	assert.Contains(t, buf.String(), "Subject: Email from Ken Thompson")
	assert.Contains(t, buf.String(), `From: ken@thompson.email`)
}

func TestMessageHTML(t *testing.T) {

	const caddyFile = `mailout {
				to              gopherHTML@domain.email
				bcc             gopherHTML1@domain.email
				subject         " HTML Email via {{ .Form.Get \"firstname\" }} {{.Form.Get \"lastname\"}}"
				body            testdata/mail_tpl.html
			}`

	buf := new(bytes.Buffer)
	srv := testMessageServer(t, caddyFile, buf, 1)
	defer srv.Close()

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("name", "Ken S. Thompson")

	testDoPost(t, srv.URL, data)

	assert.True(t, buf.Len() > 10000) // whenever you change the template, change also here
	assert.Contains(t, buf.String(), "<h3>Thank you for contacting us, Ken Thompson</h3>")
	assert.Contains(t, buf.String(), "<h3>Sir Ken S. Thompson")
	assert.Contains(t, buf.String(), "Subject: =?UTF-8?q?=EF=A3=BF_HTML_Email_via_Ken_Thompson?=")
	assert.NotContains(t, buf.String(), "Bcc: gopherHTML1@domain.email")
}

func TestMessagePlainPGPSingleKey(t *testing.T) {

	const caddyFile = `mailout {
				to              pgp@domain.email
				cc              "pgp1@domain.email"
				subject         "Encrypted contact 🔑"
				body            testdata/mail_plainTextMessage.txt
				pgp@domain.email 		testdata/B06469EE_nopw.pub.asc
			}`

	buf := new(bytes.Buffer)
	srv := testMessageServer(t, caddyFile, buf, 2)
	defer srv.Close()

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("name", "Ken Thompson")

	testDoPost(t, srv.URL, data)

	assert.Len(t, buf.String(), 2710) // whenever you change the template, change also here
	assert.Contains(t, buf.String(), "Subject: =?UTF-8?q?Encrypted_contact_=F0=9F=94=91?=")
	assert.Contains(t, buf.String(), "Cc: pgp1@domain.email")
	assert.Exactly(t, 1, bytes.Count(buf.Bytes(), maillog.MultiMessageSeparator))
	assert.Contains(t, buf.String(), `This shows the content of a text template.`)
	//t.Log(buf.String())
}

func TestMessagePlainPGPMultipleKey(t *testing.T) {

	const caddyFile = `mailout {
				to              pgp@domain.email
				cc              "pgp1@domain.email,  pgp2@domain.email"
				subject         "Encrypted contact"
				body            testdata/mail_plainTextMessage.txt
				pgp@domain.email 		testdata/B06469EE_nopw.pub.asc
				pgp1@domain.email 		testdata/6AD0EE9E_nopw.pub.asc
			}`

	buf := new(bytes.Buffer)
	srv := testMessageServer(t, caddyFile, buf, 3)
	defer srv.Close()

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("name", "Ken Thompson")

	testDoPost(t, srv.URL, data)

	assert.Len(t, buf.String(), 4957) // whenever you change the template, change also here
	assert.Exactly(t, 3, bytes.Count(buf.Bytes(), []byte("Subject: Encrypted contact")))
	assert.Exactly(t, 2, bytes.Count(buf.Bytes(), []byte(`Content-Disposition: inline; filename="encrypted.gpg"`)))
	assert.NotContains(t, buf.String(), "Cc: pgp1@domain.email")
	assert.Exactly(t, 2, bytes.Count(buf.Bytes(), maillog.MultiMessageSeparator))
	assert.Contains(t, buf.String(), `This shows the content of a text template.`)

	assert.Contains(t, buf.String(), `To: pgp@domain.email`)
	assert.Contains(t, buf.String(), `To: pgp1@domain.email`)
	assert.Contains(t, buf.String(), `Cc: pgp2@domain.email`)

	//t.Log(buf.String())
}

func TestMessagePlainText_FromEmailName(t *testing.T) {

	const caddyFile = `mailout {
				from_email		marie@gold.grimm
				from_name		"Gold Marie"
				to              brothers@fairy-tales.grimm
				subject         "Email from {{ .Form.Get \"firstname\" }} {{.Form.Get \"lastname\"}}"
				body            testdata/mail_plainTextMessage.txt
			}`

	buf := new(bytes.Buffer)
	srv := testMessageServer(t, caddyFile, buf, 1)
	defer srv.Close()

	data := make(url.Values)
	data.Set("firstname", "Marie")
	data.Set("lastname", "Pech")
	data.Set("email", "marie@pech.grimm")
	data.Set("name", "Pech Marie")

	testDoPost(t, srv.URL, data)

	assert.Len(t, buf.String(), 373) // whenever you change the template, change also here
	assert.Contains(t, buf.String(), "Email marie@pech.grimm")
	assert.Contains(t, buf.String(), `From: "Gold Marie" <marie@gold.grimm>`)
	assert.Contains(t, buf.String(), "Subject: Email from Marie Pech")
	//t.Log(buf.String())
}

func TestMessagePlainText_FromEmailOnly(t *testing.T) {

	const caddyFile = `mailout {
				from_email		marie@gold.grimm
				to              brothers@fairy-tales.grimm
				subject         "Email from {{ .Form.Get \"firstname\" }} {{.Form.Get \"lastname\"}}"
				body            testdata/mail_plainTextMessage.txt
			}`

	buf := new(bytes.Buffer)
	srv := testMessageServer(t, caddyFile, buf, 1)
	defer srv.Close()

	data := make(url.Values)
	data.Set("firstname", "Marie")
	data.Set("lastname", "Pech")
	data.Set("email", "marie@pech.grimm")
	data.Set("name", "Pech Marie")

	testDoPost(t, srv.URL, data)

	assert.Len(t, buf.String(), 358) // whenever you change the template, change also here
	assert.Contains(t, buf.String(), "Email marie@pech.grimm")
	assert.Contains(t, buf.String(), `From: marie@gold.grimm`)
	assert.Contains(t, buf.String(), "Subject: Email from Marie Pech")
	//t.Log(buf.String())
}

// 0.4.ms per PGP message
// BenchmarkMessagePlainPGP-4	    3000	    405413 ns/op	   37530 B/op	     176 allocs/op
func BenchmarkMessagePlainPGP(b *testing.B) {
	const caddyFile = `mailout {
				to              pgp@domain.email
				cc              "pgp1@domain.email"
				subject         "Encrypted contact 🔑"
				body            testdata/mail_plainTextMessage.txt
				publickey 		testdata/B06469EE_nopw.pub.asc
			}`

	c := caddy.NewTestController("http", caddyFile)
	mc, err := parse(c)
	if err != nil {
		b.Fatal(err)
	}
	if err := mc.loadTemplate(); err != nil {
		b.Fatal(err)
	}
	if err := mc.loadPGPKeys(); err != nil {
		b.Fatal(err)
	}

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("name", "Ken Thompson")

	req, err := http.NewRequest("POST", "/mailout", nil)
	if err != nil {
		b.Fatal(err)
	}

	req.PostForm = data

	buf := new(bytes.Buffer)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := newMessage(mc, req).build()
		if _, err := msg.WriteTo(buf); err != nil {
			b.Fatal(err)
		}
		buf.Reset()
	}
}
