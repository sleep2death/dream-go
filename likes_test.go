package dream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAddLike(t *testing.T) {
	testSetup()

	ctx, cancel := context.WithCancel(context.Background())
	go sdSimulating(ctx)

	defer func() {
		cancel()
		err := delUsrByName("tester010")
		if err != nil {
			t.Fatal(err)
		}
	}()

	w := testLogin(t, "tester010")
	token, c := testJwtToken(t, w)

	var n int = 2

	var res []map[string]interface{}
	// add 10 dream
	for d := 0; d < n; d++ {
		dr := newTestDream()
		dr.Prompt = dr.Prompt + " " + strconv.Itoa(n)
		req, err := postJsonReq("/api/dream/new", dr)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(token)
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)
		body := assertOK(t, w)
		res = append(res, body)

		time.Sleep(time.Millisecond * 5)
	}

	l.Debugln("waiting for 1 seconds...")
	time.Sleep(time.Second * 1)

	// add a like
	dreamId := res[0]["id"].(string)
	req, _ := http.NewRequest("GET", "/api/likes/add/"+dreamId, nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	d, err := getDreamById(dreamId)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(d.Likes))
	assert.Equal(t, c.ID, d.Likes[0])

	d, err = getDreamById(res[1]["id"].(string))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(d.Likes))

	// make exp time even shorter
	viper.SetDefault("expDreamShort", time.Second*1)
	defer viper.SetDefault("expDreamShort", time.Minute*5)

	// remove like
	req, _ = http.NewRequest("GET", "/api/likes/remove/"+dreamId, nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	// expires("d:" + dreamId)
	// wait for cache expiration
	time.Sleep(time.Millisecond * 1100)

	d, err = getDreamById(dreamId)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(d.Likes))

	// remove like again, take no effect
	req, _ = http.NewRequest("GET", "/api/likes/remove/"+dreamId, nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	// expires("d:" + dreamId)
	// wait for cache expiration
	time.Sleep(time.Millisecond * 1100)

	d, err = getDreamById(dreamId)
	assert.Nil(t, err)

	assert.Equal(t, 0, len(d.Likes))
}
