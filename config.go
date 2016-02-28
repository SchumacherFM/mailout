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

const defaultEndpoint = "/mailout"

// defaultHttpClient net/http default client does not come with a time out set.
var defaultHttpClient = &http.Client{
	Timeout: time.Second * 20,
}

type renderer interface {
	Execute(wr io.Writer, data interface{}) (err error)
}

type config struct {
	// endpoint the route name where we receive the post requests
	endpoint string

	// pgpEmailKeys a key value balanced slice containing even positions
	// the email address and on odd position the path to the PGP public key
	// path, ENV or URL [path/to/pgp.pub|https://keybase.io/cyrill/key.asc]
	// Remote keys will only be loaded from HTTP(S) sources.
	// This slice gets filled during setup.
	pgpEmailKeys []string
	// pgpEmailKeyEntities loaded and parsed publicKey. Key = email address and
	// value = public key. This map gets filled during the call to loadPGPKeys()
	pgpEmailKeyEntities map[string]*openpgp.Entity
	// pgpAttachmentName name of the email attachment file.
	pgpAttachmentName string

	// messageCount set during call to loadPGPKeys() to set number of messages
	// to create for sending.
	messageCount int

	// httpClient for now used to download an external public key
	httpClient *http.Client

	// maillog writes each email into one file in a directory. If nil, writes
	// to /dev/null also logs errors.
	maillog maillog.Logger

	// to              recipient_to@domain.email
	to []string
	// cc              recipient_cc1@domain.email, recipient_cc2@domain.email
	cc []string
	// bcc             recipient_bcc1@domain.email, recipient_bcc2@domain.email
	bcc []string
	// subject         Email from {{.firstname}} {{.lastname}}
	subject string

	// subjectTpl parsed and loaded subject template
	subjectTpl *ttpl.Template

	//body            path/to/tpl.[txt|html]
	body       string
	bodyIsHTML bool
	// bodyTpl parsed and loaded HTML or Text template for the email body.
	bodyTpl renderer

	//username        [ENV:MY_SMTP_USERNAME|gopher]
	username string
	//password        [ENV:MY_SMTP_PASSWORD|g0ph3r]
	password string
	//host            [ENV:MY_SMTP_HOST|smtp.gmail.com]
	host string
	//port            [ENV:MY_SMTP_PORT|25|587|465]
	portRaw string
	port    int

	rateLimitInterval time.Duration
	rateLimitCapacity int64
}

func newConfig() *config {
	return &config{
		endpoint:          defaultEndpoint,
		httpClient:        defaultHttpClient,
		pgpAttachmentName: "encrypted.gpg",
		host:              "localhost",
		port:              1025, // mailhog (github.com/mailhog/MailHog) default port
		rateLimitInterval: time.Hour * 24,
		rateLimitCapacity: 1000,
	}
}

// calcMessageCount calculates the number of messages to generate depending on
// the PGP key amount.
func (c *config) calcMessageCount() error {
	c.messageCount = len(c.pgpEmailKeyEntities)
	if len(c.to) > 0 || len(c.cc) > 0 || len(c.bcc) > 0 {
		c.messageCount++
	}
	return nil
}

func (c *config) loadPGPKeys() error {

	if len(c.pgpEmailKeys) == 0 {
		return nil
	}
	if l := len(c.pgpEmailKeys); l > 0 && l%2 != 0 {
		return fmt.Errorf("Imbalanced PGP email addresses and keys: %v", c.pgpEmailKeys)
	}

	c.pgpEmailKeyEntities = make(map[string]*openpgp.Entity)

	for i := 0; i < len(c.pgpEmailKeys); i = i + 2 {
		pubKey, err := c.loadPGPKey(c.pgpEmailKeys[i+1])
		if err != nil {
			return fmt.Errorf("Cannot load PGP key for email address %q with error: %s", c.pgpEmailKeys[i], err)
		}
		c.pgpEmailKeyEntities[c.pgpEmailKeys[i]] = pubKey
	}

	// remove PGP emails from to,cc and bcc
	for addr, _ := range c.pgpEmailKeyEntities {
		c.to = deleteEntrySS(c.to, addr)
		c.cc = deleteEntrySS(c.cc, addr)
		c.bcc = deleteEntrySS(c.bcc, addr)
	}
	return c.calcMessageCount()
}

func (c *config) loadPGPKey(pathToKey string) (ent *openpgp.Entity, err error) {
	var keyRC io.ReadCloser
	if strings.Index(pathToKey, "http") == 0 {
		httpData, err := c.httpClient.Get(pathToKey)
		if httpData != nil {
			keyRC = httpData.Body
			defer keyRC.Close()
		}
		if err != nil {
			return nil, fmt.Errorf("Loading of remote public key from URL %q failed:\n%s", pathToKey, err)
		}
		if httpData.StatusCode != 200 {
			return nil, fmt.Errorf("Loading remote public key failed from URL %q. StatusCode have %d StatusCode want %d", pathToKey, httpData.StatusCode, 200)
		}

	} else {
		if false == fileExists(pathToKey) {
			return nil, fmt.Errorf("File %q not found", pathToKey)
		}
		f, err := os.Open(pathToKey)
		if err != nil {
			return nil, fmt.Errorf("File %q not loaded because of error: %s", pathToKey, err)
		}
		keyRC = f
		defer keyRC.Close()
	}

	keyList, err := openpgp.ReadArmoredKeyRing(keyRC)
	if err != nil {
		return nil, fmt.Errorf("Cannot read public key %q: %s", pathToKey, err)
	}
	ent = keyList[0]

	if ent.PrivateKey != nil {
		ent = nil
		err = fmt.Errorf("PrivateKey found. Not allowed. Please remove it from resouce: %q", pathToKey)
	}
	return
}

func (c *config) loadFromEnv() error {
	var err error
	if l := len(c.pgpEmailKeys); l > 0 && l%2 == 0 {
		for i := 0; i < len(c.pgpEmailKeys); i = i + 2 {
			c.pgpEmailKeys[i+1] = loadFromEnv(c.pgpEmailKeys[i+1])
		}
	}
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
