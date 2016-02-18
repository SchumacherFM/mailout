package mailout

import (
	"testing"
	"errors"
	"net/http"
	"github.com/mholt/caddy/caddy/setup"
	"github.com/stretchr/testify/assert"
	"bufio"
	"strings"
)

var _ http.RoundTripper = (*mockTransport)(nil)

type mockTransport struct {
	resp func() *http.Response
	err  error
}

func (mt *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return mt.resp(), mt.err
}

func TestConfigLoadPGPKey(t *testing.T) {
	orgTransport := httpClient.Transport
	defer func() {
		httpClient.Transport = orgTransport
	}()

	tests := []struct {
		config       string
		expectErr    error
		keyNil       bool
		roundTripper http.RoundTripper
	}{
		{
			`mailout`,
			nil,
			true,
			nil,
		},
		{
			`mailout {
				public_key testdata/B06469EE_nopw.pub.asc
			}`,
			nil,
			false,
			nil,
		},
		{
			`mailout {
				public_key testdata/B06469EE_nopw.priv.asc
			}`,
			errors.New("PrivateKey found. Not allowed. Please remove it from file: \"testdata/B06469EE_nopw.priv.asc\""),
			true,
			nil,
		},
		{
			`mailout {
				public_key http://keybase.io/cyrill/key.asc
			}`,
			errors.New("File \"http://keybase.io/cyrill/key.asc\" not found"),
			true,
			nil,
		},
		{
			`mailout {
				public_key https://keybase.io/cyrill/keyNOTFOUND.asc
			}`,
			errors.New("File \"http://keybase.io/cyrill/key.asc\" not found"),
			true,
			&mockTransport{
				resp: func() *http.Response {
					resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader("HTTP/1.0 200 OK\r\n" +
					"Connection: close\r\n" +
					"\r\n" +
					"Body here\n")), &http.Request{Method: "GET"})
					if err != nil {
						t.Fatal(err)
					}
					return resp
				},
			},
		},
	}
	for i, test := range tests {

		c := setup.NewTestController(test.config)
		mc, err := parse(c)
		if err != nil {
			t.Fatal("Index", i, "Error:", err)
		}

		if test.roundTripper != nil {
			httpClient.Transport = test.roundTripper
		}

		err = mc.loadPGPKey()
		if test.keyNil && test.expectErr == nil {
			assert.NoError(t, err, "Index %d", i)
			assert.Nil(t, mc.keyEntity, "Index %d", i)
			continue
		}

		if test.expectErr != nil {
			assert.Nil(t, mc.keyEntity, "Index %d", i)
			assert.EqualError(t, err, test.expectErr.Error(), "Index %d", i)
			continue
		}
		assert.NoError(t, err, "Index %d", i)
		assert.NotNil(t, mc.keyEntity, "Index %d", i)
		assert.NotNil(t, mc.keyEntity.PrimaryKey, "Index %d", i)
		assert.Nil(t, mc.keyEntity.PrivateKey, "Index %d", i)
	}
}
