package mailout

import (
	"net/http"
	"testing"

	"net/http/httptest"
	"net/url"

	"fmt"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/mholt/caddy/middleware"
	"github.com/stretchr/testify/assert"
)

func newTestHandler(t *testing.T, caddyFile string) *handler {
	c := setup.NewTestController(caddyFile)
	mc, err := parse(c)
	if err != nil {
		t.Fatal(err)
	}
	h := newHandler(mc, nil)
	h.Next = middleware.HandlerFunc(func(w http.ResponseWriter, r *http.Request) (int, error) {
		return http.StatusTeapot, nil
	})
	return h
}

func TestServeHTTP_ShouldNotValidateEmailAddress(t *testing.T) {
	t.Parallel()

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
	assert.Exactly(t, http.StatusOK, code)
	assert.Exactly(t, `{"error":"Invalid email address: \"ken\\uf8ffthompson.email\""}`+"\n", w.Body.String())
	assert.Exactly(t, StatusUnprocessableEntity, w.Code)
	assert.Exactly(t, HeaderApplicationJSONUTF8, w.HeaderMap.Get(HeaderContentType))
}

func TestServeHTTP_ShouldNotParseForm(t *testing.T) {
	t.Parallel()

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
	assert.Exactly(t, http.StatusBadRequest, code)
	assert.EqualError(t, err, "missing form body")
	assert.Empty(t, w.HeaderMap)
}

func TestServeHTTP_ShouldNotMatchPOST(t *testing.T) {
	t.Parallel()

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
	assert.Exactly(t, http.StatusMethodNotAllowed, code)
	assert.Empty(t, w.HeaderMap)
}

func TestServeHTTP_ShouldNotMatchEndpoint(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	h := newTestHandler(t, `mailout {
		ratelimit_interval 3s
		ratelimit_capacity 5
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
		assert.Exactly(t, http.StatusOK, code, "Request %d", i)

		//t.Log(HeaderXRateLimitLimit, w.HeaderMap.Get(HeaderXRateLimitLimit))
		assert.Exactly(t, fmt.Sprintf("%d", i), w.HeaderMap.Get(HeaderXRateLimitLimit))

		// t.Log(HeaderXRateLimitRemaining, w.HeaderMap.Get(HeaderXRateLimitRemaining))
		assert.Exactly(t, fmt.Sprintf("%d", 5-i), w.HeaderMap.Get(HeaderXRateLimitRemaining))

		//t.Log(HeaderXRateLimitReset, w.HeaderMap.Get(HeaderXRateLimitReset))
		assert.Exactly(t, "3", w.HeaderMap.Get(HeaderXRateLimitReset))
		//t.Log("Request",i,"\n")
		assert.Exactly(t, HeaderApplicationJSONUTF8, w.HeaderMap.Get(HeaderContentType))
		assert.Len(t, w.HeaderMap, 4)
	}

	{
		w := httptest.NewRecorder()
		code, err := h.ServeHTTP(w, req)
		if err != nil {
			t.Fatal("Request:", 6, "Error:", err)
		}
		assert.Exactly(t, http.StatusOK, code)
		assert.Exactly(t, http.StatusTooManyRequests, w.Code)
		assert.Exactly(t, "6", w.HeaderMap.Get(HeaderXRateLimitLimit))
		assert.Exactly(t, "-1", w.HeaderMap.Get(HeaderXRateLimitRemaining))
		assert.Exactly(t, "2", w.HeaderMap.Get(HeaderXRateLimitReset))
		assert.Len(t, w.HeaderMap, 3)
	}
}
