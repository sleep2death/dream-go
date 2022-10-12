package dream

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

type dreamStatus int

const (
	dsPending dreamStatus = iota
	dsProcessing
	dsDone
	dsFailed
	dsNsfw
)

type dream struct {
	ID     string  `json:"_id" bson:"_id"`
	Prompt string  `json:"prompt" bson:"prompt"`
	Steps  int     `json:"steps" bson:"steps"`
	Scale  float32 `json:"scale" bson:"scale"`
	Width  int     `json:"width" bson:"width"`
	Height int     `json:"height" bson:"height"`
	Seed   int64   `json:"seed" bson:"seed"`

	// following data will be generated at server side
	Author   string      `json:"author" bson:"author"`
	AuthorID string      `json:"authorId" bson:"authorId"`
	Status   dreamStatus `json:"status" bson:"status"`
	Images   []string    `json:"image" bson:"image"`

	Created  time.Time `json:"created" bson:"created"`
	Finished time.Time `json:"finished" bson:"finished"`
}

func dreamHandlers() {
	r.POST("/api/dream/new", jwtAuth, newDreamHandler)
	r.GET("/api/dream/status/:id", jwtAuth, dreamStatusHandler)
}

// create a new dream
func newDreamHandler(c *gin.Context) {

	d := &dream{}

	// bind the data
	// l.Debugln("dream bind before:", d)
	err := c.ShouldBind(d)

	// l.Debugln("dream bind after:", d)
	if err != nil {
		badRequest(c, errors.New("dream.invalid.params"))
		return
	}

	d.ID = uuid.New().String()
	d.Status = dsPending
	d.Created = time.Now()
	d.Author = c.GetString("username") // add author name by http-only cookie
	d.AuthorID = c.GetString("uuid")   // add author id by http-only cookie

	l.Debugln("new dream:", d)

	err = addDream(d) // insert it into mongodb, then cache it with redis
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"id": d.ID,
	})
}

// get dream status
func dreamStatusHandler(c *gin.Context) {
	dreamId := c.Param("id")
	if len(dreamId) == 0 {
		badRequest(c, errors.New("dream.invalid.params"))
	}

	d, err := getDreamById(dreamId)
	if err == redis.Nil || err == mongo.ErrNoDocuments {
		badRequest(c, errors.New("dream.invalid.notFound"))
	} else if err != nil {
		internalError(c, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"status": d.Status,
	})
}
