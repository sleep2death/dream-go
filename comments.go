package dream

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func commentsHandlers() {
	r.POST("/api/comments/add/:dreamId", jwtAuth, addCommentHandler)
}

type comment struct {
	ID      string    `json:"_id" bson:"_id"`
	Text    string    `json:"text" bson:"text"`
	Author  string    `json:"author" bson:"author"`
	Dream   string    `json:"dream" bson:"dream"`
	Created time.Time `json:"created" bson:"created"`
	Likes   []like    `json:"likes" bson:"likes"`
}

func addCommentHandler(c *gin.Context) {
	text := strings.Trim(c.PostForm("text"), " ")

	if len(text) == 0 || len(text) > viper.GetInt("commentMaxLen") {
		badRequest(c, errors.New("comment.invalid.length"))
		return
	}

	dreamId := c.Param("dreamId")
	if len(dreamId) == 0 {
		badRequest(c, errors.New("comment.invalid.dreamId"))
		return
	}

	co := comment{
		ID:      uuid.New().String(),
		Text:    text,
		Author:  c.GetString("uuid"),
		Dream:   dreamId,
		Created: time.Now(),
		Likes:   make([]like, 0),
	}

	_, err := comments.InsertOne(context.TODO(), co)
	if err != nil {
		internalError(c, err)
		return
	}

	ok(c)
}

// get comments by dream ID
func getCommentsById(dreamId string, pageIdx int) ([]comment, error) {
	var cs []comment
	err := getCache("d:"+dreamId+":comments:"+strconv.Itoa(pageIdx), &cs)

	// if the dream already cached
	if err == nil {
		l.Debugln("comments cached by redis:", dreamId)
		return cs, nil
	}

	// if error occcurs, ant it's not nil
	if err != nil && err != redis.Nil {
		return nil, err
	}

	// clear cache
	err = nil

	cpp := int64(viper.GetInt("commentsPerPage"))
	skip := cpp * int64(pageIdx)

	opts := options.Find().SetLimit(cpp).SetSkip(skip)

	ctx := context.TODO()
	cursor, err := comments.Find(ctx, bson.M{"dream": dreamId}, opts)
	if err != nil {
		return nil, err
	}

	// get all comments
	err = cursor.All(ctx, &cs)
	if err != nil {
		return nil, err
	}

	l.Debugln("find comments of the dream:", dreamId, cs)

	// set cache
	if len(cs) > 0 {
		err = setCache("d:"+dreamId+":comments:"+strconv.Itoa(pageIdx), cs, viper.GetDuration("expComments"))
		if err != nil {
			return nil, err
		}
	}

	return cs, nil
}
