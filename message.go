package mailout

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"

	"strings"

	"github.com/SchumacherFM/mailout/bufpool"
	"github.com/SchumacherFM/mailout/maillog"
	"golang.org/x/crypto/openpgp"
	"gopkg.in/gomail.v2"
)

// pgpStartText is a marker which denotes the end of the message and the start of
// an armored signature.
var pgpStartText = []byte("-----BEGIN PGP SIGNATURE-----\n\n")

// pgpEndText is a marker which denotes the end of the armored signature.
var pgpEndText = []byte("\n-----END PGP SIGNATURE-----")

type message struct {
	mc *config
	r  *http.Request
}

type messages []*gomail.Message

func (ms messages) WriteTo(w io.Writer) (sum int64, err error) {
	for j, m := range ms {
		var i int64
		i, err = m.WriteTo(w)
		if err != nil {
			return
		}
		sum += i
		if j < len(ms)-1 {
			var i2 int
			i2, err = w.Write(maillog.MultiMessageSeparator)
			if err != nil {
				return
			}
			sum += int64(i2)
		}
	}
	return
}

// newMessage uses also a request which must have an already parsed form.
func newMessage(mc *config, r *http.Request) message {
	return message{
		mc: mc,
		r:  r,
	}
}

func (bm message) build() messages {

	msgs := bm.initMessages()

	i := 0
	// build all encrypted emails
	for addr := range bm.mc.pgpEmailKeyEntities {
		msg := msgs[i]
		msg.SetHeader("To", addr)
		bm.setFrom(msg)
		bm.renderSubject(msg)
		bm.bodyEncrypted(msg, addr)
		i++
	}

	// build all non-encrypted emails
	// private information leakage if some recipients use PGP keys and others not.
	for ; i < len(msgs); i++ {
		msg := msgs[i]
		bm.setNonPGPRecipients(msg)
		bm.setFrom(msg)
		bm.renderSubject(msg)
		bm.bodyUnencrypted(msg)
	}

	return msgs
}

// initMessages creates a slice with non-nil message pointers
func (bm message) initMessages() (msgs messages) {
	msgs = make(messages, bm.mc.messageCount)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = gomail.NewMessage()
	}
	return
}

func (bm message) setNonPGPRecipients(gm *gomail.Message) {
	if len(bm.mc.to) > 0 {
		gm.SetHeader("To", bm.mc.to...)
	}
	if len(bm.mc.cc) > 0 {
		gm.SetHeader("Cc", bm.mc.cc...)
	}
	if len(bm.mc.bcc) > 0 {
		gm.SetHeader("Bcc", bm.mc.bcc...)
	}
}

func (bm message) setFrom(gm *gomail.Message) {
	if bm.mc.fromEmail != "" && bm.mc.fromName != "" {
		gm.SetAddressHeader("From", bm.mc.fromEmail, bm.mc.fromName)
		return
	}
	if bm.mc.fromEmail != "" {
		gm.SetHeader("From", bm.mc.fromEmail)
		return
	}

	if n := strings.TrimSpace(bm.r.PostFormValue("name")); n != "" {
		gm.SetAddressHeader("From", bm.r.PostFormValue("email"), n)
		return
	}
	gm.SetHeader("From", bm.r.PostFormValue("email"))
}

func (bm message) renderSubject(gm *gomail.Message) {
	subjBuf := bufpool.Get()
	defer bufpool.Put(subjBuf)

	err := bm.mc.subjectTpl.Execute(subjBuf, struct {
		Form    url.Values
		Request *http.Request
	}{
		Form:    bm.r.PostForm,
		Request: bm.r,
	})
	if err != nil {
		bm.mc.maillog.Errorf("Render Subject Error: %s\nForm: %#v\nWritten: %s", err, bm.r.PostForm, subjBuf)
	}
	gm.SetHeader("Subject", subjBuf.String())
}

func (bm message) bodyEncrypted(gm *gomail.Message, pgpTo string) {

	pgpBuf := bufpool.Get()
	defer bufpool.Put(pgpBuf)

	msgBuf := bufpool.Get()
	defer bufpool.Put(msgBuf)

	bm.renderTemplate(msgBuf)

	// the next line may crash if the PGP key gets removed ... some how. but the crash is fine
	w, err := openpgp.Encrypt(pgpBuf, openpgp.EntityList{0: bm.mc.pgpEmailKeyEntities[pgpTo]}, nil, nil, nil)
	if err != nil {
		bm.mc.maillog.Errorf("PGP encrypt Error: %s", err)
		return
	}

	_, err = w.Write(msgBuf.Bytes())
	if err != nil {
		bm.mc.maillog.Errorf("PGP encrypt Write Error: %s", err)
		return
	}

	err = w.Close()
	if err != nil {
		bm.mc.maillog.Errorf("PGP encrypt Close Error: %s", err)
		return
	}

	b64Buf := make([]byte, base64.StdEncoding.EncodedLen(pgpBuf.Len()))
	base64.StdEncoding.Encode(b64Buf, pgpBuf.Bytes())

	gm.SetBody("text/plain", "This should be an OpenPGP/MIME encrypted message (RFC 4880 and 3156)")

	gm.Embed(
		bm.mc.pgpAttachmentName,
		gomail.SetCopyFunc(func(w io.Writer) error {
			if _, err := w.Write(pgpStartText); err != nil {
				return err
			}
			if _, err := w.Write(b64Buf); err != nil {
				return err
			}
			if _, err := w.Write(pgpEndText); err != nil {
				return err
			}
			return nil
		}),
	)
}

func (bm message) bodyUnencrypted(gm *gomail.Message) {
	contentType := "text/plain"
	if bm.mc.bodyIsHTML {
		contentType = "text/html"
	}

	buf := bufpool.Get()
	defer bufpool.Put(buf)

	bm.renderTemplate(buf)
	gm.SetBody(contentType, buf.String())
}

func (bm message) renderTemplate(buf *bytes.Buffer) {
	err := bm.mc.bodyTpl.Execute(buf, struct {
		Form    url.Values
		Request *http.Request
	}{
		Form:    bm.r.PostForm,
		Request: bm.r,
	})
	if err != nil {
		bm.mc.maillog.Errorf("Render Error: %s\nForm: %#v\nWritten: %s", err, bm.r.PostForm, buf)
	}
}
