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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGetFeed(t *testing.T) {
	testSetup()

	ctx, cancel := context.WithCancel(context.Background())
	go sdSimulating(ctx)

	defer func() {
		cancel()
		err := delUsrByName("tester004")
		if err != nil {
			t.Fatal(err)
		}
	}()

	w := testLogin(t, "tester004")
	token, c := testJwtToken(t, w)

	var n int = 10

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
		assertOK(t, w)

		time.Sleep(time.Millisecond * 5)
	}

	l.Debugln("waiting for 1 seconds...")
	time.Sleep(time.Second * 1)

	// all dreams were processed
	usr, err := getUserById(c.ID)
	assert.Nil(t, err)
	assert.Equal(t, n, len(usr.Outbox))

	// get feed
	req, err := http.NewRequest("GET", "/api/feeds/get", nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body := assertOK(t, w)
	feeds := body["feeds"].([]interface{})
	assert.Equal(t, n, len(feeds))

	// feed will be empty when get feed again
	req, err = http.NewRequest("GET", "/api/feeds/get", nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	assert.Nil(t, body["feeds"])

	// remove a seen dream from cache, then it will get one
	err = rdb.SPop(context.TODO(), "u:"+c.ID+":seen").Err()
	assert.Nil(t, err)

	req, err = http.NewRequest("GET", "/api/feeds/get", nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	feeds = body["feeds"].([]interface{})
	assert.Equal(t, 1, len(feeds))

	// add 5 more dreams
	n = 5
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
		assertOK(t, w)

		time.Sleep(time.Millisecond * 5)
	}

	l.Debugln("waiting for 1 seconds...")
	time.Sleep(time.Second * 1)

	// set the limit of feeds
	viper.Set("feedLimit", 3)

	// feeds will be 3 when get feed again
	req, err = http.NewRequest("GET", "/api/feeds/new", nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	feeds = body["feeds"].([]interface{})
	assert.Equal(t, 3, len(feeds))

	req, err = http.NewRequest("GET", "/api/feeds/get", nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	feeds = body["feeds"].([]interface{})
	assert.Equal(t, 3, len(feeds))

	// two left
	req, err = http.NewRequest("GET", "/api/feeds/new", nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)
	feeds = body["feeds"].([]interface{})
	assert.Equal(t, 2, len(feeds))

	// set-back
	viper.Set("feedLimit", 16)

	// remove a seen dream from cache, then it will get one
	dreamId, err := rdb.SPop(context.TODO(), "u:"+c.ID+":seen").Result()
	assert.Nil(t, err)

	// update the dream's generated time to 5 days agao
	_, err = dreams.UpdateOne(context.TODO(), bson.M{"_id": dreamId}, bson.M{"$set": bson.M{
		"finished": primitive.NewDateTimeFromTime(time.Now().AddDate(0, 0, -5)),
	}}, nil)

	assert.Nil(t, err)

	res, err := accounts.UpdateOne(context.TODO(), bson.M{"_id": c.ID, "outbox.dream": dreamId}, bson.M{"$set": bson.M{
		"outbox.$.generated": primitive.NewDateTimeFromTime(time.Now().AddDate(0, 0, -5)),
	}})

	assert.Nil(t, err)
	assert.Equal(t, int64(1), res.ModifiedCount)

	// clear cache
	expires("d:" + dreamId)
	expires("u:" + c.ID)
	expires("u:" + c.ID + ":feed:new")

	req, err = http.NewRequest("GET", "/api/feeds/get", nil)
	req.AddCookie(token)
	w = httptest.NewRecorder()

	r.ServeHTTP(w, req)
	body = assertOK(t, w)

	// filter out one by time, and two left
	feeds = body["feeds"].([]interface{})
	assert.Equal(t, 2, len(feeds))
}
