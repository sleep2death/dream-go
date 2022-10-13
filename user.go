package dream

import "time"

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

	Likes []like `json:"likes" bson:"likes"` // dreams which user liked
}
