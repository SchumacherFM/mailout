package mailout

import (
	"fmt"
	htpl "html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	ttpl "text/template"
	"time"

	"github.com/SchumacherFM/mailout/maillog"
	"golang.org/x/crypto/openpgp"
	"gopkg.in/gomail.v2"
)

const emailSplitBy = ","
const emailPublicKeyAttachmentName = "encrypted.gpg"
const defaultEndpoint = "/mailout"

var defaultHttpClient = &http.Client{
	Timeout: time.Second * 20,
}

type renderer interface {
	Execute(wr io.Writer, data interface{}) (err error)
}

type config struct {
	// endpoint the route name where we receive the post requests
	endpoint string

	// publicKey path, ENV or URL [path/to/pgp.pub|https://keybase.io/cyrill/key.asc]
	// only loads from https
	publicKey string
	// primaryKey loaded and parsed publicKey
	keyEntity  *openpgp.Entity
	httpClient *http.Client
	// keyAttachmentName name of the email attachment file.
	keyAttachmentName string

	// maillog writes each email into one file in a directory. If nil, writes to /dev/null
	maillog *maillog.Logger

	//to              recipient_to@domain.email
	to []string
	//cc              recipient_cc1@domain.email, recipient_cc2@domain.email
	cc []string
	//bcc             recipient_bcc1@domain.email, recipient_bcc2@domain.email
	bcc []string
	//subject         Email from {{.firstname}} {{.lastname}}
	subject string

	subjectTpl *ttpl.Template

	//body            path/to/tpl.[txt|html]
	body       string
	bodyIsHTML bool
	bodyTpl    renderer

	//username        [ENV:MY_SMTP_USERNAME|gopher]
	username string
	//password        [ENV:MY_SMTP_PASSWORD|g0ph3r]
	password string
	//host            [ENV:MY_SMTP_HOST|smtp.gmail.com]
	host string
	//port            [ENV:MY_SMTP_PORT|25|587|465]
	portRaw string
	port    int
}

func newConfig() *config {
	return &config{
		endpoint:          defaultEndpoint,
		httpClient:        defaultHttpClient,
		keyAttachmentName: emailPublicKeyAttachmentName,
		host:              "localhost",
		port:              1025, // mailcatcher (a ruby app) default port
	}
}

func (c *config) loadPGPKey() error {

	if c.publicKey == "" {
		return nil
	}

	var keyRC io.ReadCloser
	if strings.Index(c.publicKey, "https://") == 0 {
		httpData, err := c.httpClient.Get(c.publicKey)
		if httpData != nil {
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

	keyList, err := openpgp.ReadArmoredKeyRing(keyRC)
	if err != nil {
		return fmt.Errorf("Cannot read public key %q: %s", c.publicKey, err)
	}
	c.keyEntity = keyList[0]

	if c.keyEntity.PrivateKey != nil {
		c.keyEntity = nil
		return fmt.Errorf("PrivateKey found. Not allowed. Please remove it from file: %q", c.publicKey)
	}

	return nil
}

func (c *config) loadFromEnv() error {
	var err error
	c.publicKey = loadFromEnv(c.publicKey)
	c.username = loadFromEnv(c.username)
	c.password = loadFromEnv(c.password)
	c.host = loadFromEnv(c.host)
	c.portRaw = loadFromEnv(c.portRaw)
	c.port, err = strconv.Atoi(c.portRaw)
	return err
}

func (c *config) pingSMTP() error {
	d := gomail.NewPlainDialer(c.host, c.port, c.username, c.password)
	sc, err := d.Dial()
	if err != nil {
		return err
	}
	return sc.Close()
}

func (c *config) loadTemplate() (err error) {
	if false == fileExists(c.body) {
		return fmt.Errorf("File %q not found", c.body)
	}

	switch filepath.Ext(c.body) {
	case ".txt":
		c.bodyTpl, err = ttpl.ParseFiles(c.body)
	case ".html":
		c.bodyIsHTML = true
		c.bodyTpl, err = htpl.ParseFiles(c.body)
	}

	if c.bodyTpl == nil && err == nil {
		return fmt.Errorf("Incorrect file extension. Neither .txt nor .html: %q", c.body)
	}

	c.subjectTpl, err = ttpl.New("").Parse(c.subject)
	return
}

func loadFromEnv(s string) string {
	const envPrefix = `ENV:`
	if strings.Index(s, envPrefix) != 0 {
		return s
	}
	return os.Getenv(s[len(envPrefix):])
}

func splitEmailAddresses(s string) ([]string, error) {
	ret := strings.Split(s, emailSplitBy)
	for i, val := range ret {
		ret[i] = strings.TrimSpace(val)
		if false == isValidEmail(ret[i]) {
			return nil, fmt.Errorf("Incorrect Email address found in: %q", s)
		}
	}
	return ret, nil
}
