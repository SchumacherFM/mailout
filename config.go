package mailout

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

var httpClient = &http.Client{
	Timeout: time.Second * 20,
	// for testing we can exchange Transport with a mock
}

type config struct {
	// publicKey path or URL [path/to/pgp.pub|https://keybase.io/cyrill/key.asc]
	// only loads from https
	publicKey string
	// primaryKey loaded and parsed publicKey
	keyEntity *openpgp.Entity

	// logDir a path to log directory for errors and data log.
	// won't start without a existing directory
	logDir string

	// successUri redirects to this URL after posting the data
	successUri string

	//to              recipient_to@domain.email
	to string
	//cc              recipient_cc1@domain.email, recipient_cc2@domain.email
	cc string
	//bcc             recipient_bcc1@domain.email, recipient_bcc2@domain.email
	bcc string
	//subject         Email from {{.firstname}} {{.lastname}}
	subject string
	//body            path/to/tpl.[txt|html]
	body string

	//username        [ENV:MY_SMTP_USERNAME|gopher]
	username string
	//password        [ENV:MY_SMTP_PASSWORD|g0ph3r]
	password string
	//host            [ENV:MY_SMTP_HOST|smtp.gmail.com]
	host string
	//port            [ENV:MY_SMTP_PORT|25|587|465]
	port int
}

func (c *config) loadPGPKey() error {

	if c.publicKey == "" {
		return nil
	}

	var keyRC io.ReadCloser
	if strings.Index(c.publicKey, "https://") == 0 {
		httpData, err := httpClient.Get(c.publicKey)
		if httpData.Body != nil {
			keyRC = httpData.Body
			defer keyRC.Close()
		}
		if err != nil {
			return fmt.Errorf("Loading of remote public key from URL %q failed:\n%s", c.publicKey, err)
		}
		if httpData.StatusCode != 200 {
			return fmt.Errorf("Loading remote public key failed from URL %q. StatusCode have %d StatusCode want %d", c.publicKey, httpData.StatusCode, 200)
		}

	} else {
		if false == fileExists(c.publicKey) {
			return fmt.Errorf("File %q not found", c.publicKey)
		}
		f, err := os.Open(c.publicKey)
		if err != nil {
			return fmt.Errorf("File %q not loaded because of error: %s", c.publicKey, err)
		}
		keyRC = f
		defer keyRC.Close()
	}

	var err error
	c.keyEntity, err = openpgp.ReadEntity(packet.NewReader(keyRC))
	if err != nil {
		return fmt.Errorf("Cannot read public key %q: %s", c.publicKey, err)
	}

	return nil
}

func loadFromEnv(s string) string {
	const envPrefix = `ENV:`
	if strings.Index(s,envPrefix) != 0 {
		return s
	}
	// os.Getenv()
}

// IsDir returns true if path is a directory
func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return fileInfo != nil && fileInfo.IsDir() && err == nil
}

// fileExists returns true if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
