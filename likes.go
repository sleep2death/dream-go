package dream

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
)

type like struct {
	Author string `json:"author" bson:"author"`
	// Added  time.Time `json:"added" bson:"added"`
}

func likesHandlers() {
	r.GET("/api/likes/add/:dreamId", jwtAuth, addLikeHandler)
	r.GET("/api/likes/remove/:dreamId", jwtAuth, removeLikeHandler)
}

func addLikeHandler(c *gin.Context) {
	uuid := c.GetString("uuid")
	dreamId := c.Param("dreamId")

	if len(dreamId) == 0 {
		badRequest(c, errors.New("invalid.input"))
		return
	}

	err := addLike(uuid, dreamId)
	if err != nil {
		internalError(c, err)
		return
	}

	ok(c)
}

func removeLikeHandler(c *gin.Context) {
	uuid := c.GetString("uuid")
	dreamId := c.Param("dreamId")

	if len(dreamId) == 0 {
		badRequest(c, errors.New("invalid.input"))
		return
	}

	err := removeLike(uuid, dreamId)
	if err != nil {
		internalError(c, err)
		return
	}

	ok(c)
}

// addLike to "dream" with "author"
func addLike(author string, dream string) error {
	// TODO: cache it or using msessage queue
	ctx := context.TODO()
	res, err := dreams.UpdateByID(ctx, dream, bson.M{
		"$addToSet": bson.M{"likes": author},
	})

	if err != nil {
		return err
	}

	l.Debugln("add like from", dream, "by", author, ":", res.ModifiedCount)

	// make the cache expires in a short time
	expiresIn("d:"+dream, viper.GetDuration("expDreamShort"))
	return nil
}

func removeLike(author string, dream string) error {
	// TODO: cache it or using msessage queue
	ctx := context.TODO()
	res, err := dreams.UpdateByID(ctx, dream, bson.M{
		"$pull": bson.M{"likes": author},
	})

	if err != nil {
		return err
	}

	l.Debugln("remove like from", dream, "by", author, ":", res.ModifiedCount)

	// make the cache expires in a short time
	expiresIn("d:"+dream, viper.GetDuration("expDreamShort"))
	return nil
}
