package mailout

import (
	"crypto/tls"
	"net/http"
	"time"

	"gopkg.in/gomail.v2"
)

func startMailDaemon(mc *config) chan<- *http.Request {
	rChan := make(chan *http.Request)
	// this can be a bottleneck under high load because the channel is unbuffered.
	// maybe we can add a pool of sendmail workers.
	go goMailDaemonRecoverable(mc, rChan)
	return rChan
}

// goMailDaemonRecoverable self restarting goroutine.
// TODO(cs) limit restarting to e.g. 10 tries and then crash it.
func goMailDaemonRecoverable(mc *config, rChan <-chan *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			mc.maillog.Errorf("[mailout] Catching panic %#v and restarting daemon ...", r)
			go goMailDaemonRecoverable(mc, rChan)
		}
	}()
	goMailDaemon(mc, rChan)
}

func goMailDaemon(mc *config, rChan <-chan *http.Request) {
	d := gomail.NewPlainDialer(mc.host, mc.port, mc.username, mc.password)
	if mc.port == 587 {
		d.TLSConfig = &tls.Config{
			ServerName: mc.host, // host names must match between this one and the one requested in the cert.
		}
	}
	if mc.skipTlsVerify {
		if d.TLSConfig == nil {
			d.TLSConfig = &tls.Config{}
		}
		d.TLSConfig.InsecureSkipVerify = true
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

			mails := newMessage(mc, r).build()
			// multiple mails will increase the rate limit at some MTAs.
			// so the REST API rate limit must be: rate / pgpEmailAddresses

			if !open {
				if s, err = d.Dial(); err != nil {
					mc.maillog.Errorf("Dial Error: %s", err)

					wc := mc.maillog.NewWriter()
					if _, errW := mails.WriteTo(wc); errW != nil {
						mc.maillog.Errorf("Dial: Message WriteTo Log Error: %s", errW)
					}
					if errC := wc.Close(); errC != nil {
						mc.maillog.Errorf("Dial wc.Close Error: %s", errC)
					}

					continue
				}
				open = true
			}

			wc := mc.maillog.NewWriter()
			if _, err2 := mails.WriteTo(wc); err2 != nil {
				mc.maillog.Errorf("Send: Message WriteTo Log Error: %s", err)
			}
			if err = wc.Close(); err != nil {
				mc.maillog.Errorf("Send wc.Close Error: %s", err)
			}

			if err := gomail.Send(s, mails...); err != nil {
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
			//default: // http://www.jtolds.com/writing/2016/03/go-channels-are-bad-and-you-should-feel-bad/
		}
	}
}
