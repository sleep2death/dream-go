package dream

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database

var users *mongo.Collection
var dreams *mongo.Collection
var comments *mongo.Collection

var ErrInvalidPwd = errors.New("invalid password")

func ensureIndeces() {
	// Ensure indeces for users
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
	}
	if _, err := users.Indexes().CreateMany(
		context.TODO(),
		models,
	); err != nil {
		panic(err)
	}

	// Ensure indeces for dreams
	models = []mongo.IndexModel{
		{Keys: bson.D{{Key: "author", Value: 1}}},
		{Keys: bson.D{{Key: "created", Value: 1}}},
	}
	if _, err := dreams.Indexes().CreateMany(
		context.TODO(),
		models,
	); err != nil {
		panic(err)
	}

	// Ensure indeces for comments
	if _, err := comments.Indexes().CreateMany(
		context.TODO(),
		models,
	); err != nil {
		panic(err)
	}
}

func newUser(email string, username string, hashedPassword string) error {
	id := uuid.New().String()

	usr := &user{
		ID:      id,
		Email:   email,
		Name:    username,
		HPwd:    hashedPassword,
		Created: time.Now(),

		Following: []string{}, // follow user self first
		Followers: []string{},

		Outbox: make([]feed, 0),
		Inbox:  make([]feed, 0),
	}

	res, err := users.InsertOne(context.TODO(), usr)

	if err != nil {
		return err
	}
	l.Infoln("ADD_USER", username, res.InsertedID)
	return nil
}

func findUsrByName(name string, project bson.M) (usr bson.M, err error) {
	opts := options.FindOne()
	opts.SetProjection(project) // only password needed

	err = users.FindOne(context.TODO(), bson.D{{Key: "username", Value: name}}, opts).Decode(&usr)
	return
}

func findUsrByEmail(email string, project bson.M) (usr bson.M, err error) {
	opts := options.FindOne()
	opts.SetProjection(project) // only password needed

	err = users.FindOne(context.TODO(), bson.D{{Key: "email", Value: email}}, opts).Decode(&usr)
	return
}

func delUsrByName(name string) error {
	match := bson.M{
		"username": name,
	}
	res, err := users.DeleteOne(context.TODO(), match)
	if err != nil {
		return err
	}
	if res.DeletedCount > 0 {
		l.Infoln("DEL_USER", name)
	}
	return nil
}
