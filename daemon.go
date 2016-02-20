package mailout

import (
	"crypto/tls"
	"net/http"
	"time"

	"gopkg.in/gomail.v2"
)

func startMailDaemon(mc *config) chan<- *http.Request {
	rChan := make(chan *http.Request)

	go func() {
		d := gomail.NewPlainDialer(mc.host, mc.port, mc.username, mc.password)
		if mc.port == 587 {
			d.TLSConfig = &tls.Config{
				ServerName: mc.host,
			}
		}

		var s gomail.SendCloser
		var err error
		open := false
		for {
			select {
			case r, ok := <-rChan:
				if !ok {
					return
				}

				msg := newMessage(mc, r).build()

				if !open {
					if s, err = d.Dial(); err != nil {
						mc.maillog.Errorf("Dial Error: %s", err)

						wc := mc.maillog.NewWriter()
						if _, errW := msg.WriteTo(wc); errW != nil {
							mc.maillog.Errorf("Dial: Message WriteTo Log Error: %s\nMessage: %#v", errW, msg)
						}
						if errC := wc.Close(); errC != nil {
							mc.maillog.Errorf("Dial wc.Close Error: %s", errC)
						}

						continue
					}
					open = true
				}

				wc := mc.maillog.NewWriter()
				if _, err2 := msg.WriteTo(wc); err2 != nil {
					mc.maillog.Errorf("Send: Message WriteTo Log Error: %s\nMessage: %#v", err, msg)
				}
				if err = wc.Close(); err != nil {
					mc.maillog.Errorf("Send wc.Close Error: %s", err)
				}

				if err := gomail.Send(s, msg); err != nil {
					mc.maillog.Errorf("Send Error: %s", err)
				}

			// Close the connection to the SMTP server if no email was sent in
			// the last 30 seconds.
			case <-time.After(30 * time.Second):
				if open {
					if err := s.Close(); err != nil {
						mc.maillog.Errorf("Dial Close Error: %s", err)
					}
					open = false
				}
			}
		}
	}()

	return rChan
}
