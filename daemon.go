package mailout

import (
	"log"
	"net/http"
	"time"

	"gopkg.in/gomail.v2"
)

func startMailDaemon(mc *config) chan<- *http.Request {
	rChan := make(chan *http.Request)

	go func() {
		d := gomail.NewPlainDialer(mc.host, mc.port, mc.username, mc.password)

		var s gomail.SendCloser
		var err error
		open := false
		for {
			select {
			case r, ok := <-rChan:
				if !ok {
					return
				}
				if !open {
					if s, err = d.Dial(); err != nil {
						panic(err) // todo remove
						// add exponential backoff strategy
					}
					open = true
				}

				msg := newMessage(mc, r).build()

				if err := gomail.Send(s, msg); err != nil {
					log.Println(err)
				}
			// Close the connection to the SMTP server if no email was sent in
			// the last 30 seconds.
			case <-time.After(30 * time.Second):
				if open {
					if err := s.Close(); err != nil {
						log.Println(err)
					}
					open = false
				}
			}
		}
	}()

	return rChan
}
