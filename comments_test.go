package dream

import (
	"context"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAddComment(t *testing.T) {
	testSetup()

	ctx, cancel := context.WithCancel(context.Background())
	go sdSimulating(ctx)

	defer func() {
		cancel()
		err := delUsrByName("tester011")
		if err != nil {
			t.Fatal(err)
		}

		err = delUsrByName("tester012")
		if err != nil {
			t.Fatal(err)
		}
	}()

	w := testLogin(t, "tester011")
	tokenA, _ := testJwtToken(t, w)

	w = testLogin(t, "tester012")
	tokenB, _ := testJwtToken(t, w)

	var n int = 3
	var res []map[string]interface{}

	// add 3 dreams
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
		body := assertOK(t, w)
		res = append(res, body)

		time.Sleep(time.Millisecond * 5)
	}

	l.Debugln("waiting for 500 ms...")
	time.Sleep(time.Millisecond * 500)

	dreamId := res[0]["id"].(string)

	n = 20
	for c := 0; c < n; c++ {
		req, err := postFormReq("/api/comments/add/"+dreamId, map[string]string{
			"text": "hello, this is a comment",
		})
		if err != nil {
			t.Fatal(err)
		}

		req.AddCookie(tokenB)
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assertOK(t, w)
		time.Sleep(time.Millisecond * 10)
	}

	// comments page ONE
	comments, err := getCommentsById(dreamId, 0)
	assert.Nil(t, err)
	assert.Equal(t, viper.GetInt("commentsPerPage"), len(comments))

	// comments page TWO
	comments, err = getCommentsById(dreamId, 1)
	assert.Nil(t, err)
	assert.Equal(t, n-viper.GetInt("commentsPerPage"), len(comments))
}
