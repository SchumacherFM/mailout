package mailout

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"

	"github.com/SchumacherFM/mailout/bufpool"
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
	gm *gomail.Message
}

func newMessage(mc *config, r *http.Request) message {
	return message{
		mc: mc,
		r:  r,
		gm: gomail.NewMessage(),
	}
}

func (bm message) build() *gomail.Message {
	bm.header()
	if bm.mc.keyEntity != nil {
		bm.bodyEncrypted()
	} else {
		bm.bodyUnencrypted()
	}
	return bm.gm
}

func (bm message) header() {
	bm.gm.SetHeader("To", bm.mc.to...)
	if len(bm.mc.cc) > 0 {
		bm.gm.SetHeader("Cc", bm.mc.cc...)
	}
	if len(bm.mc.bcc) > 0 {
		bm.gm.SetHeader("Bcc", bm.mc.bcc...)
	}
	bm.gm.SetHeader("Subject", bm.mc.subject)

	bm.gm.SetAddressHeader("From", bm.r.PostFormValue("email"), bm.r.PostFormValue("name"))

}

func (bm message) bodyEncrypted() {

	pgpBuf := bufpool.Get()
	defer bufpool.Put(pgpBuf)

	msgBuf := bufpool.Get()
	defer bufpool.Put(msgBuf)

	bm.renderTemplate(msgBuf)

	w, err := openpgp.Encrypt(pgpBuf, openpgp.EntityList{0: bm.mc.keyEntity}, nil, nil, nil)
	if err != nil {
		panic(err) // todo remove
	}
	_, err = w.Write(msgBuf.Bytes())
	if err != nil {
		panic(err) // todo remove
	}
	err = w.Close()
	if err != nil {
		panic(err) // todo remove
	}

	b64Buf := make([]byte, base64.StdEncoding.EncodedLen(pgpBuf.Len()))
	base64.StdEncoding.Encode(b64Buf, pgpBuf.Bytes())

	bm.gm.SetBody("text/plain", "This should be an OpenPGP/MIME encrypted message (RFC 4880 and 3156)")

	bm.gm.Embed(
		"encrypted.asc",
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

func (bm message) bodyUnencrypted() {
	contentType := "text/plain"
	if bm.mc.bodyIsHTML {
		contentType = "text/html"
	}
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	bm.renderTemplate(buf)
	bm.gm.SetBody(contentType, buf.String())
}

func (bm message) renderTemplate(buf *bytes.Buffer) {
	err := bm.mc.bodyTpl.Execute(buf, struct {
		Form url.Values
	}{
		Form: bm.r.PostForm,
	})
	if err != nil {
		bm.mc.maillog.Errorf("Render Error: %s\nForm: %#v\nWritten: %s", err, bm.r.PostForm, buf)
	}
}
