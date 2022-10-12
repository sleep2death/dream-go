package dream

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type feed struct {
	Dream     string    `json:"dream" bson:"dream"`         // dream "id"
	Generated time.Time `json:"generated" bson:"generated"` // generated time
}

func feedHandlers() {
	r.GET("/api/feeds/get", jwtAuth, feedsGetHandler)
	r.GET("/api/feeds/new", jwtAuth, feedsNewHandler)
}

// get user's feeds
func feedsGetHandler(c *gin.Context) {
	uuid := c.GetString("uuid")
	if len(uuid) == 0 {
		badRequest(c, errors.New("user id not found"))
		return
	}

	s := time.Now().AddDate(0, 0, viper.GetInt("feedUpdatedLimit"))
	feeds, err := getFeeds(uuid, s)

	if err != nil {
		internalError(c, err)
		return
	}

	// l.Debugln("feeds length internal", len(feeds))
	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"feeds": feeds,
	})
}

func feedsNewHandler(c *gin.Context) {
	uuid := c.GetString("uuid")
	if len(uuid) == 0 {
		badRequest(c, errors.New("user id not found"))
		return
	}

	since := time.Now().AddDate(0, 0, viper.GetInt("feedUpdatedLimit"))
	feeds, err := hasNewFeeds(uuid, since)

	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"feeds": feeds,
	})
}
