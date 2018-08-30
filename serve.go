package mailout

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/SchumacherFM/mailout/bufpool"
	"github.com/gorilla/sessions"
	"github.com/juju/ratelimit"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/quasoft/memstore"
	"github.com/steambap/captcha"
)

// StatusUnprocessableEntity gets returned whenever parsing of the form fails.
const StatusUnprocessableEntity = 422

// StatusEmpty returned by mailout middleware because the proper status gets
// written previously
const StatusEmpty = 0

const (
	headerContentType         = "Content-Type"
	headerApplicationJSONUTF8 = "application/json; charset=utf-8"
	headerPNG                 = "image/png"
)

type ReCaptchaResp struct {
	Success     bool     `json:"success"`
	ChallengeTs string    `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

var MailSessionsStore = memstore.NewMemStore(
	[]byte("authkey123"),
	[]byte("enckey12341234567890123456789012"),
)

func newHandler(mc *config, mailPipe chan<- *http.Request) *handler {
	return &handler{
		rlBucket: ratelimit.NewBucket(mc.rateLimitInterval, mc.rateLimitCapacity),
		reqPipe:  mailPipe,
		config:   mc,
	}
}

type handler struct {
	// rlBucket rate limit bucket
	rlBucket *ratelimit.Bucket
	// reqPipe send request to somewhere else. can be nil for testing.
	reqPipe chan<- *http.Request
	config  *config
	Next    httpserver.Handler
}

// ServeHTTP serves a request
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	var err error
	var session *sessions.Session

	// captcha
	if h.config.Captcha {
		if r.URL.Path == h.config.endpoint+"/captcha" {
			data, _ := captcha.New(150, 50)
			session, err = MailSessionsStore.New(r, "captcha")
			if err != nil {
				return h.writeJSON(JSONError{
					Code:  http.StatusAccepted,
					Error: err.Error(),
				}, w)
			}
			session.Values["captcha"] = data.Text
			err = MailSessionsStore.Save(r, w, session)
			if err != nil {
				return h.writeJSON(JSONError{
					Code:  http.StatusAccepted,
					Error: err.Error(),
				}, w)
			}
			w.Header().Set(headerContentType, headerPNG)
			return http.StatusOK, data.WriteImage(w)
		}
	}

	if r.URL.Path != h.config.endpoint {
		return h.Next.ServeHTTP(w, r)
	}

	if r.Method != "POST" {
		return h.writeJSON(JSONError{
			Code:  http.StatusMethodNotAllowed,
			Error: http.StatusText(http.StatusMethodNotAllowed),
		}, w)
	}

	if _, ok := h.rlBucket.TakeMaxDuration(1, h.config.rateLimitInterval); !ok {
		return h.writeJSON(JSONError{
			Code:  http.StatusTooManyRequests,
			Error: http.StatusText(http.StatusTooManyRequests),
		}, w)
	}

	if err := r.ParseForm(); err != nil {
		return h.writeJSON(JSONError{
			Code:  http.StatusBadRequest,
			Error: err.Error(),
		}, w)
	}

	// captcha
	if h.config.Captcha {
		session, err = MailSessionsStore.Get(r, "captcha")
		if err != nil {
			return h.writeJSON(JSONError{
				Code:  http.StatusAccepted,
				Error: err.Error(),
			}, w)
		}
		text := r.PostFormValue("captcha_text")
		if text != session.Values["captcha"] {
			session.Values["captcha"] = ""
			MailSessionsStore.Save(r, w, session)
			return h.writeJSON(JSONError{
				Code:  http.StatusAccepted,
				Error: "Wrong captcha_text: " + text + " Correct: " + session.Values["captcha"].(string),
			}, w)
		}
	}

	// recaptcha
	if h.config.ReCaptcha {
		RecaptchaText := r.PostFormValue("g-recaptcha-response")
		httpclient := http.Client{}
		parts := url.Values{}
		parts.Set("secret", h.config.ReCaptchaSecret)
		parts.Set("response", RecaptchaText)
		parts.Set("remoteip", r.RemoteAddr)
		r, err := httpclient.PostForm("https://www.google.com/recaptcha/api/siteverify", parts)
		if err != nil {
			return h.writeJSON(JSONError{
				Code:  http.StatusAccepted,
				Error: err.Error(),
			}, w)
		}
		resp := &ReCaptchaResp{}
		err = json.NewDecoder(r.Body).Decode(resp)
		if err != nil {
			return h.writeJSON(JSONError{
				Code:  http.StatusAccepted,
				Error: err.Error(),
			}, w)
		}
		if resp.Success != true {
			return h.writeJSON(JSONError{
				Code:  http.StatusAccepted,
				Error: strings.Join(resp.ErrorCodes, "; "),
			}, w)
		}
	}

	if e := r.PostFormValue("email"); !isValidEmail(e) {
		return h.writeJSON(JSONError{
			Code:  StatusUnprocessableEntity,
			Error: fmt.Sprintf("Invalid email address: %q", e),
		}, w)
	}

	// captcha
	if h.config.Captcha {
		session.Values["captcha"] = ""
		MailSessionsStore.Save(r, w, session)
	}

	if h.reqPipe != nil {
		h.reqPipe <- r // might block if the mail daemon is busy
	}
	return h.writeJSON(JSONError{Code: http.StatusOK}, w)
}

// JSONError defines how an REST JSON looks like.
// Code 200 and empty Error specifies a successful request
// Any other Code value s an error.
type JSONError struct {
	// Code represents the HTTP Status Code, a work around.
	Code int `json:"code,omitempty"`
	// Error the underlying error, if there is one.
	Error string `json:"error,omitempty"`
}

func (h *handler) writeJSON(je JSONError, w http.ResponseWriter) (int, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	w.Header().Set(headerContentType, headerApplicationJSONUTF8)

	// https://github.com/mholt/caddy/issues/637#issuecomment-189599332
	w.WriteHeader(je.Code)

	if err := json.NewEncoder(buf).Encode(je); err != nil {
		return http.StatusInternalServerError, err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return http.StatusInternalServerError, err
	}

	return StatusEmpty, nil
}
