package dream

import "github.com/gin-gonic/gin"

type like struct {
}

func likesHandlers() {
	r.GET("/api/likes/add", jwtAuth, addLikeHandler)
	r.GET("/api/likes/remove", jwtAuth, removeLikeHandler)
}

func addLikeHandler(c *gin.Context) {

}

func removeLikeHandler(c *gin.Context) {

}
