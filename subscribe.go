package dream

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func subscribeHandlers() {
	r.GET("/api/sub/:id", jwtAuth, subscribeHandler)
	r.GET("/api/unsub/:id", jwtAuth, unsubscribeHandler)
}

func subscribeHandler(c *gin.Context) {
	uuid := c.GetString("uuid")
	if len(uuid) == 0 {
		badRequest(c, errors.New("user id not found"))
		return
	}

	if len(c.Param("id")) == 0 {
		badRequest(c, errors.New("following id not found"))
		return
	}

	err := addFollowing(uuid, c.Param("id"))
	if err != nil {
		internalError(c, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})
}

func unsubscribeHandler(c *gin.Context) {
	uuid := c.GetString("uuid")
	if len(uuid) == 0 {
		badRequest(c, errors.New("user id not found"))
		return
	}

	if len(c.Param("id")) == 0 {
		badRequest(c, errors.New("following id not found"))
		return
	}

	err := removeFollowing(uuid, c.Param("id"))
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})
}
