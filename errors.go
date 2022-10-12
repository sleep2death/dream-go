package dream

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"ok":  false,
		"msg": err.Error(),
	})
	l.Errorln("bad request", err)
}

func internalError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"ok":  false,
		"msg": "internal.error", // don't send the error to client
	})
	l.Errorln("internal server error", err)
}

func permissionError(c *gin.Context, err error) {
	c.JSON(http.StatusMethodNotAllowed, gin.H{
		"ok":  false,
		"msg": err.Error(),
	})
	l.Errorln("method not allowed", err)
}
