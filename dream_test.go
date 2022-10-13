package dream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
)

func newTestDream() *dream {
	return &dream{
		Prompt: "Hello, World!",
		Steps:  51,
		Scale:  7.5,
		Width:  512,
		Height: 512,
		Seed:   1024,
	}
}

func sdSimulating(ctx context.Context) {
	timeout := time.Second * 10 // 10 seconds

	// l.Debugln("stable diffusion canceled")
	for {
		select {
		case <-ctx.Done():
			l.Debugln("stable diffusion canceled")
			return
		default:
			l.Debugln("stable diffusion start fetching job")
			res, err := rdb.BLPop(ctx, timeout, "DQ").Result()
			if err != nil && err != redis.Nil && err != context.Canceled { // if something wrong
				l.Fatal(err)
				// l.Debugln("queue failed", err)
			} else if err == redis.Nil { // if not found, continue
				continue
			} else if err == context.Canceled {
				return
			}

			dreamId := res[1]

			d, err := getDreamById(dreamId)
			if err != nil {
				l.Panic(err, dreamId)
				// l.Debugln("queue failed", err)
			}

			// update dream status
			d.Status = dsProcessing
			err = updateDream(d, false)
			if err != nil {
				// l.Debugln("queue failed", err)
				l.Panic(err)
			}

			// simulating stable diffusion processing
			time.Sleep(time.Millisecond * 50)

			// update dream status again
			d.Status = dsDone
			size := strconv.Itoa(d.Width) + "x" + strconv.Itoa(d.Height)
			img := d.ID + "_" + size
			d.Images = []string{img + "_origin", img + "_thumb", img + "_square"}

			now := time.Now()
			d.Finished = now

			err = updateDream(d, false)
			if err != nil {
				l.Panic(err)
			}

			// push dream to user's outbox
			err = addFeed(d)
			if err != nil {
				l.Panic(err)
			}
		}
	}
}

func TestNewDream(t *testing.T) {
	testSetup()

	defer func() {
		err := delUsrByName("tester")
		if err != nil {
			t.Fatal(err)
		}
	}()

	w := testLogin(t, "tester")
	token, _ := testJwtToken(t, w)

	req, err := postJsonReq("/api/dream/new", newTestDream())
	if err != nil {
		t.Fatal(err)
	}

	req.AddCookie(token)
	// l.Infoln("test new dream:", req)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body := assertOK(t, w)

	id := body["id"].(string)
	l.Debugln("new dream id:", id)

	_, err = getDreamById(id)
	assert.Nil(t, err)

	// clear dream's cache
	expires("d:" + id)
	_, err = getDreamById(id)
	assert.Nil(t, err)
}

func TestDreamStatus(t *testing.T) {
	testSetup()

	defer func() {
		err := delUsrByName("tester")
		if err != nil {
			t.Fatal(err)
		}
	}()

	// new user
	w := testLogin(t, "tester")
	token, _ := testJwtToken(t, w)

	// new dream
	req, err := postJsonReq("/api/dream/new", newTestDream())
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body := assertOK(t, w)

	id := body["id"].(string)
	l.Debugln("new dream id:", id)

	// get dream status
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/api/dream/status/"+id, nil)
	req.AddCookie(token)

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	assert.Equal(t, body["status"], float64(dsPending))
}

func TestDreamUpdate(t *testing.T) {
	testSetup()

	ctx, cancel := context.WithCancel(context.Background())
	go sdSimulating(ctx)

	defer func() {
		err := delUsrByName("tester003")
		if err != nil {
			t.Fatal(err)
		}
		cancel()
	}()

	// new user
	w := testLogin(t, "tester003")
	token, c := testJwtToken(t, w)

	// 10 new dreams
	var n int = 10
	for d := 0; d < n; d++ {
		req, err := postJsonReq("/api/dream/new", newTestDream())
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(token)
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assertOK(t, w)

		time.Sleep(time.Millisecond * 5)
	}

	l.Debugln("waiting for 2 seconds...")
	time.Sleep(time.Second * 2)

	// all dreams were processed
	usr, err := getUserById(c.ID)
	assert.Nil(t, err)
	assert.Equal(t, n, len(usr.Outbox))

	// get from redis cache
	usr, err = getUserById(c.ID)
	assert.Nil(t, err)
	assert.Equal(t, n, len(usr.Outbox))

	// create another dream
	req, err := postJsonReq("/api/dream/new", newTestDream())
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assertOK(t, w)

	time.Sleep(time.Millisecond * 100)
	usr, err = getUserById(c.ID)

	assert.Nil(t, err)
	assert.Equal(t, n+1, len(usr.Outbox))
}
