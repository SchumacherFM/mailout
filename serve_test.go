package mailout

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/stretchr/testify/assert"
)

func newTestHandler(t *testing.T, caddyFile string) *handler {
	c := caddy.NewTestController("http", caddyFile)
	mc, err := parse(c)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(mc, nil)
	h.Next = httpserver.HandlerFunc(func(w http.ResponseWriter, r *http.Request) (int, error) {
		return http.StatusTeapot, nil
	})
	return h
}

func TestServeHTTP_ShouldNotValidateEmailAddress(t *testing.T) {

	h := newTestHandler(t, `mailout`)

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "kenthompson.email")
	data.Set("name", "Ken Thompson")

	req, err := http.NewRequest("POST", "/mailout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.PostForm = data

	w := httptest.NewRecorder()
	code, err := h.ServeHTTP(w, req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, StatusEmpty, code)
	assert.Exactly(t, "{\"code\":422,\"error\":\"Invalid email address: \\\"ken\\\\uf8ffthompson.email\\\"\"}\n", w.Body.String())
	assert.Exactly(t, StatusUnprocessableEntity, w.Code)
	assert.Exactly(t, headerApplicationJSONUTF8, w.HeaderMap.Get(headerContentType))
}

func TestServeHTTP_ShouldNotParseForm(t *testing.T) {

	h := newTestHandler(t, `mailout {
		ratelimit_interval 3s
		ratelimit_capacity 5
	}`)
	req, err := http.NewRequest("POST", "/mailout", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	code, err := h.ServeHTTP(w, req)
	assert.Exactly(t, StatusEmpty, code)
	assert.Exactly(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, err)
	assert.Len(t, w.HeaderMap, 1)
}

func TestServeHTTP_ShouldNotMatchPOST(t *testing.T) {

	h := newTestHandler(t, `mailout {
		ratelimit_interval 3s
		ratelimit_capacity 5
	}`)
	req, err := http.NewRequest("GET", "/mailout", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	code, err := h.ServeHTTP(w, req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, StatusEmpty, code)
	assert.Exactly(t, http.StatusMethodNotAllowed, w.Code)
	assert.Len(t, w.HeaderMap, 1)
}

func TestServeHTTP_ShouldNotMatchEndpoint(t *testing.T) {

	h := newTestHandler(t, `mailout /hiddenMailService {
		ratelimit_interval 3s
		ratelimit_capacity 5
	}`)
	req, err := http.NewRequest("POST", "/mailout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.PostForm = make(url.Values)
	w := httptest.NewRecorder()
	code, err := h.ServeHTTP(w, req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, http.StatusTeapot, code)
	assert.Empty(t, w.HeaderMap)
}

func TestServeHTTP_RateLimitShouldBeApplied(t *testing.T) {

	h := newTestHandler(t, `mailout {
		ratelimit_interval 100ms
		ratelimit_capacity 4
	}`)

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("name", "Ken Thompson")

	req, err := http.NewRequest("POST", "/mailout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.PostForm = data

	for i := 1; i <= 5; i++ {
		w := httptest.NewRecorder()
		code, err := h.ServeHTTP(w, req)
		if err != nil {
			t.Fatal("Request:", i, "Error:", err)
		}
		assert.Exactly(t, StatusEmpty, code, "Request %d", i)
		assert.Exactly(t, http.StatusOK, w.Code, "Request %d", i)

		//t.Log("Request",i,"\n")
		assert.Exactly(t, headerApplicationJSONUTF8, w.HeaderMap.Get(headerContentType))
		assert.Len(t, w.HeaderMap, 1)
	}

	for i := 6; i <= 8; i++ {
		w := httptest.NewRecorder()
		code, err := h.ServeHTTP(w, req)
		if err != nil {
			t.Fatal("Request:", i, "Error:", err)
		}
		assert.Exactly(t, StatusEmpty, code, "Request %d", i)
		assert.Exactly(t, http.StatusTooManyRequests, w.Code, "Request %d", i)

		assert.Len(t, w.HeaderMap, 1, "Request %d", i)
	}

	i := 9
	time.Sleep(time.Millisecond * 100)
	w := httptest.NewRecorder()
	code, err := h.ServeHTTP(w, req)
	if err != nil {
		t.Fatal("Request:", i, "Error:", err)
	}
	assert.Exactly(t, StatusEmpty, code, "Request %d", i)
	assert.Exactly(t, http.StatusOK, w.Code, "Request %d", i)
	assert.Len(t, w.HeaderMap, 1, "Request %d", i)
}

func TestServeHTTP_ShouldRedirectToGivenURL(t *testing.T) {

	h := newTestHandler(t, `mailout {
		redirect_field redirect_to
	}`)

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("redirect_to", "http://foo.com/bar")

	req, err := http.NewRequest("POST", "/mailout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.PostForm = data

	w := httptest.NewRecorder()
	code, err := h.ServeHTTP(w, req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, StatusEmpty, code)
	assert.Exactly(t, http.StatusSeeOther, w.Code)
	assert.Exactly(t, "http://foo.com/bar", w.HeaderMap.Get("Location"))
}

func TestServeHTTP_ShouldNotRedirectWithoutRedirectionField(t *testing.T) {

	h := newTestHandler(t, `mailout {
		redirect_field redirect_to
	}`)

	data := make(url.Values)
	data.Set("firstname", "Ken")
	data.Set("lastname", "Thompson")
	data.Set("email", "ken@thompson.email")
	data.Set("redirect_url", "http://foo.com/bar")		// wrong field name!

	req, err := http.NewRequest("POST", "/mailout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.PostForm = data

	w := httptest.NewRecorder()
	code, err := h.ServeHTTP(w, req)
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, StatusEmpty, code)
	assert.Exactly(t, http.StatusOK, w.Code)
	assert.Exactly(t, "", w.HeaderMap.Get("Location"))
}
