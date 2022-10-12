package dream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSubscribe(t *testing.T) {
	testSetup()

	defer func() {
		err := delUsrByName("tester006")
		if err != nil {
			t.Fatal(err)
		}

		err = delUsrByName("tester007")
		if err != nil {
			t.Fatal(err)
		}

		err = delUsrByName("tester008")
		if err != nil {
			t.Fatal(err)
		}

		err = delUsrByName("tester009")
		if err != nil {
			t.Fatal(err)
		}
	}()

	w := testLogin(t, "tester006")
	tokenA, cA := testJwtToken(t, w)

	w = testLogin(t, "tester007")
	tokenB, cB := testJwtToken(t, w)

	w = testLogin(t, "tester008")
	tokenC, cC := testJwtToken(t, w)
	// time.Sleep(time.Millisecond * 10)

	w = testLogin(t, "tester009")
	tokenD, cD := testJwtToken(t, w)

	// A subs B
	req, _ := http.NewRequest("GET", "/api/sub/"+cB.ID, nil)
	req.AddCookie(tokenA)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	// B subs A
	req, _ = http.NewRequest("GET", "/api/sub/"+cA.ID, nil)
	req.AddCookie(tokenB)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	// C subs A
	req, _ = http.NewRequest("GET", "/api/sub/"+cA.ID, nil)
	req.AddCookie(tokenC)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	// D subs B
	req, _ = http.NewRequest("GET", "/api/sub/"+cB.ID, nil)
	req.AddCookie(tokenD)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	// expires user's cache
	// expires("u:" + cA.ID)
	// expires("u:" + cB.ID)
	// expires("u:" + cC.ID)

	// user A
	usr, err := getUserById(cA.ID)
	assert.Nil(t, err)

	assert.Contains(t, usr.Following, cB.ID)
	assert.Contains(t, usr.Followers, cB.ID)
	assert.Contains(t, usr.Followers, cC.ID)

	// user B
	usr, err = getUserById(cB.ID)
	assert.Nil(t, err)

	assert.Contains(t, usr.Following, cA.ID)
	assert.Contains(t, usr.Followers, cA.ID)

	// user D
	usr, err = getUserById(cD.ID)
	assert.Nil(t, err)

	assert.Equal(t, usr.Following[0], cB.ID)

	// add background job
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sdSimulating(ctx)

	var n int = 10

	// A created 10 dreams
	for d := 0; d < n; d++ {
		dr := newTestDream()
		dr.Prompt = dr.Prompt + " " + strconv.Itoa(n)
		req, err := postJsonReq("/api/dream/new", dr)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(tokenA)
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assertOK(t, w)

		time.Sleep(time.Millisecond * 5)
	}

	l.Debugln("waiting for 1 seconds for processing dreams...")
	time.Sleep(time.Second * 1)

	// B will get new feeds
	req, err = http.NewRequest("GET", "/api/feeds/new", nil)
	req.AddCookie(tokenB)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body := assertOK(t, w)
	feeds := body["feeds"].([]interface{})
	assert.Equal(t, n, len(feeds))

	// C will get new feeds
	req, err = http.NewRequest("GET", "/api/feeds/new", nil)
	req.AddCookie(tokenC)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	feeds = body["feeds"].([]interface{})
	assert.Equal(t, n, len(feeds))

	// D will NOT get new feeds, when A created dreams
	req, err = http.NewRequest("GET", "/api/feeds/new", nil)
	req.AddCookie(tokenD)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	assert.Nil(t, body["feeds"])

	// C will get 10 new feeds from cache
	req, err = http.NewRequest("GET", "/api/feeds/get", nil)
	req.AddCookie(tokenC)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	feeds = body["feeds"].([]interface{})
	assert.Equal(t, n, len(feeds))

	// B unsub A
	req, _ = http.NewRequest("GET", "/api/unsub/"+cA.ID, nil)
	req.AddCookie(tokenB)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	expires("u:" + cB.ID)
	usr, err = getUserById(cB.ID)

	assert.Nil(t, err)
	assert.NotContains(t, usr.Following, cA.ID)

	expires("u:" + cA.ID)
	usr, err = getUserById(cA.ID)

	assert.Nil(t, err)
	assert.NotContains(t, usr.Followers, cB.ID)

	n = 3

	// A created 3 dreams
	for d := 0; d < n; d++ {
		dr := newTestDream()
		dr.Prompt = dr.Prompt + " " + strconv.Itoa(n)
		req, err := postJsonReq("/api/dream/new", dr)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(tokenA)
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assertOK(t, w)

		time.Sleep(time.Millisecond * 5)
	}

	expires("u:" + cB.ID + ":feed:new")
	expires("u:" + cC.ID + ":feed:new")

	l.Debugln("waiting for 1 seconds for processing dreams...")
	time.Sleep(time.Second * 1)

	usr, err = getUserById(cB.ID)
	assert.Nil(t, err)
	assert.Zero(t, len(usr.Following))
	// l.Debugln("B's following", usr.Following)

	req, err = http.NewRequest("GET", "/api/feeds/new", nil)
	req.AddCookie(tokenB)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	l.Debugln("body is", body)
	// B will not receive new feeds from A
	assert.Nil(t, body["feeds"])

	req, err = http.NewRequest("GET", "/api/feeds/new", nil)
	req.AddCookie(tokenC)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	feeds = body["feeds"].([]interface{})
	// C will receive new feeds from A
	assert.Equal(t, n, len(feeds))
}
