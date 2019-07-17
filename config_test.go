package mailout

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/caddyserver/caddy"
	"github.com/stretchr/testify/assert"
)

var _ http.RoundTripper = (*mockTransport)(nil)

type mockTransport struct {
	Transport http.RoundTripper
	URL       *url.URL
}

func (mt mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// note that url.URL.ResolveReference doesn't work here
	// since t.u is an absolute url
	req.URL.Scheme = mt.URL.Scheme
	req.URL.Host = mt.URL.Host
	req.URL.Path = path.Join(mt.URL.Path, req.URL.Path)
	rt := mt.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(req)
}

func mockServerTransport(code int, body string) func() (*httptest.Server, http.RoundTripper) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, body)
	}))

	srvURL, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	tr := mockTransport{URL: srvURL}

	return func() (*httptest.Server, http.RoundTripper) { return server, tr }
}

func TestConfigLoadPGPKeyHTTPS(t *testing.T) {

	tests := []struct {
		config     string
		expectErr  error
		keyNil     bool
		serverMock func() (*httptest.Server, http.RoundTripper)
		msgCount   int
	}{
		{
			`mailout {
				go@ogle.com https://keybase.io/cyrill/keyNOTFOUND.asc
			}`,
			errors.New("[mailout] Cannot load PGP key for email address \"go@ogle.com\" with error: [mailout] Loading remote public key failed from URL \"https://keybase.io/cyrill/keyNOTFOUND.asc\". StatusCode have 404 StatusCode want 200"),
			true,
			mockServerTransport(http.StatusNotFound, "Not found"),
			0,
		},
		{
			`mailout {
				go@ogle.com https://keybase.io/cyrill/B06469EE_nopw.pub.asc
			}`,
			errors.New("[mailout] Cannot load PGP key for email address \"go@ogle.com\" with error: [mailout] Cannot read public key \"https://keybase.io/cyrill/B06469EE_nopw.pub.asc\": openpgp: invalid argument: no armored data found"),
			true,
			mockServerTransport(http.StatusOK, "I'm hacking ..."),
			0,
		},
		{
			`mailout {
				go@ogle.com https://keybase.io/cyrill/B06469EE_nopw.pub.asc
			}`,
			nil,
			false,
			mockServerTransport(http.StatusOK, testPubKey),
			1,
		},
	}
	for i, test := range tests {

		srv, trsp := test.serverMock()

		c := caddy.NewTestController("http", test.config)
		mc, err := parse(c)
		if err != nil {
			srv.Close()
			t.Fatal("Index", i, "Error:", err)
		}

		mc.httpClient.Transport = trsp

		err = mc.loadPGPKeys()
		assert.Exactly(t, test.msgCount, mc.messageCount, "Index %d", i)
		srv.Close()
		if test.keyNil && test.expectErr == nil {
			assert.NoError(t, err, "Index %d", i)
			assert.Empty(t, mc.pgpEmailKeyEntities, "Index %d", i)
			continue
		}

		if test.expectErr != nil {
			assert.Empty(t, mc.pgpEmailKeyEntities, "Index %d", i)
			assert.EqualError(t, err, test.expectErr.Error(), "Index %d", i)
			continue
		}
		assert.NoError(t, err, "Index %d", i)
		assert.NotNil(t, mc.pgpEmailKeyEntities, "Index %d", i)
		assert.NotNil(t, mc.pgpEmailKeyEntities["go@ogle.com"].PrimaryKey, "Index %d", i)
		assert.Nil(t, mc.pgpEmailKeyEntities["go@ogle.com"].PrivateKey, "Index %d", i)
	}
}

func TestConfigLoadPGPKeyHDD(t *testing.T) {

	tests := []struct {
		config    string
		expectErr error
		keyNil    bool
	}{
		{
			`mailout`,
			nil,
			true,
		},
		{
			`mailout {
				go@ogle.com testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			false,
		},
		{
			`mailout {
				go@ogle.com testdata/B06469EE_nopw.priv.asc
			}`,
			errors.New("[mailout] Cannot load PGP key for email address \"go@ogle.com\" with error: [mailout] PrivateKey found. Not allowed. Please remove it from resouce: \"testdata/B06469EE_nopw.priv.asc\""),
			true,
		},
		{
			`mailout {
				go@ogle.com xhttp://keybase.io/cyrill/key.asc
			}`,
			errors.New("[mailout] Cannot load PGP key for email address \"go@ogle.com\" with error: File \"xhttp://keybase.io/cyrill/key.asc\" not found"),
			true,
		},
	}
	for i, test := range tests {

		c := caddy.NewTestController("http", test.config)
		mc, err := parse(c)
		if err != nil {
			t.Fatal("Index", i, "Error:", err)
		}

		err = mc.loadPGPKeys()
		if test.keyNil && test.expectErr == nil {
			assert.NoError(t, err, "Index %d", i)
			assert.Empty(t, mc.pgpEmailKeyEntities, "Index %d", i)
			continue
		}

		if test.expectErr != nil {
			assert.Empty(t, mc.pgpEmailKeyEntities, "Index %d", i)
			assert.EqualError(t, err, test.expectErr.Error(), "Index %d", i)
			continue
		}
		assert.NoError(t, err, "Index %d", i)
		assert.NotNil(t, mc.pgpEmailKeyEntities, "Index %d", i)
		assert.NotNil(t, mc.pgpEmailKeyEntities["go@ogle.com"].PrimaryKey, "Index %d", i)
		assert.Nil(t, mc.pgpEmailKeyEntities["go@ogle.com"].PrivateKey, "Index %d", i)
	}
}

const testPubKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----
Comment: GPGTools - http://gpgtools.org

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
V4sIaARfoWRiSvBACooywFQwjpbuPIc=
=M5xL
-----END PGP PUBLIC KEY BLOCK-----
`

func TestLoadFromEnv(t *testing.T) {

	const testCaddyConfig = `mailout {
	pgpmail@domain.host		ENV:CADDY_MAILOUT_KEY
	username				ENV:CADDY_MAILOUT_USER
	password				ENV:CADDY_MAILOUT_PW
	host            		ENV:CADDY_MAILOUT_HOST
	port            		1030
}`

	assert.NoError(t, os.Setenv("CADDY_MAILOUT_KEY", "testdata/B06469EE_nopw.pub.asc"))
	assert.NoError(t, os.Setenv("CADDY_MAILOUT_USER", "luser"))
	assert.NoError(t, os.Setenv("CADDY_MAILOUT_PW", "123456"))
	assert.NoError(t, os.Setenv("CADDY_MAILOUT_HOST", "127.0.0.4"))

	wantConfig := newConfig()
	wantConfig.pgpEmailKeys = []string{`pgpmail@domain.host`, `testdata/B06469EE_nopw.pub.asc`}
	wantConfig.username = "luser"
	wantConfig.password = "123456"
	wantConfig.host = "127.0.0.4"
	wantConfig.portRaw = "1030"
	wantConfig.port = 1030
	wantConfig.messageCount = 0

	c := caddy.NewTestController("http", testCaddyConfig)
	mc, err := parse(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := mc.loadFromEnv(); err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, wantConfig, mc)
}

func TestLoadTemplate(t *testing.T) {

	tests := []struct {
		caddyfile string
		wantErr   error
	}{
		{
			`mailout {
				body            testdata/mail_tpl_NOTFOUND.html
			}`,
			errors.New("[mailout] File \"testdata/mail_tpl_NOTFOUND.html\" not found"),
		},
		{
			`mailout {
				body            testdata/mail_tpl.phtml
			}`,
			errors.New("[mailout] Incorrect file extension. Neither .txt nor .html: \"testdata/mail_tpl.phtml\""),
		},
		{
			`mailout {
				body            testdata/mail_tpl.html
			}`,
			nil,
		},
		{
			`mailout {
				body            testdata/mail_tpl.txt
			}`,
			nil,
		},
	}
	for i, test := range tests {
		c := caddy.NewTestController("http", test.caddyfile)
		mc, err := parse(c)
		if err != nil {
			t.Fatal(err)
		}

		tplErr := mc.loadTemplate()

		if test.wantErr != nil {
			assert.Nil(t, mc.bodyTpl)
			assert.EqualError(t, tplErr, test.wantErr.Error(), "Index %d ", i)
			continue
		}
		assert.NoError(t, tplErr, "Index %d ", i)
		assert.NotNil(t, mc.bodyTpl, "Index %d ", i)
	}
}

func TestPingSMTP_OK(t *testing.T) {

	if os.Getenv("MAILOUT_MAILCATCHER") == "" {
		t.Skip("Please set env variable MAILOUT_MAILCATCHER to test pingSMTP on your local machine.")
	}

	c := newConfig()
	assert.Nil(t, c.pingSMTP())
}

func TestPingSMTP_Fail(t *testing.T) {

	if os.Getenv("MAILOUT_MAILCATCHER") == "" {
		t.Skip("Please set env variable MAILOUT_MAILCATCHER to test pingSMTP on your local machine.")
	}

	c := newConfig()
	c.port = 4711
	assert.EqualError(t, c.pingSMTP(), "dial tcp [::1]:4711: getsockopt: connection refused")
}
