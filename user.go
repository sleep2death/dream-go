package dream

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type user struct {
	ID      string    `json:"_id" bson:"_id"`
	Name    string    `json:"username" bson:"username"`
	Email   string    `json:"email" bson:"email"`
	HPwd    string    `json:"password" bson:"password"` // hashed password
	Created time.Time `json:"created" bson:"created"`   // created time

	Following []string `json:"following" bson:"following"` // subscriptions of the user
	Followers []string `json:"followers" bson:"followers"` // subscriptions of the user

	Outbox  []feed    `json:"outbox" bson:"outbox"`   // subscriptions of the user
	Inbox   []feed    `json:"inbox" bson:"inbox"`     // subscriptions of the user
	Updated time.Time `json:"updated" bson:"updated"` // created time

	Likes       []like `json:"likes" bson:"likes"` // dreams which user liked
	Initialized bool   `json:"init" bson:"init"`
}

func userHandlers() {
	r.GET("/api/user/is_new", wxJwtAuth, isUserNewHandler)
}

func isUserNewHandler(c *gin.Context) {
	usr, err := getUserById(c.GetString("id"), "init")
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":   true,
		"init": usr.Initialized,
	})
}
