package dream

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v9"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

var (
	rdb *redis.Client      // redis client
	mdb *mongo.Client      // mongodb client
	l   *zap.SugaredLogger // logger
	r   *gin.Engine        // gin's router
)

// Setup the server
func Setup(engine *gin.Engine) {
	// create zap logger instance
	var logger *zap.Logger
	if viper.GetString("mode") == "release" {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync() // flushes buffer, if any
	l = logger.Sugar()

	// connect to redis and mongodb
	rdb, mdb = dbConn()

	// set router
	r = engine

	// setup handlers
	pingHandlers()      // ping handlers
	authHandlers()      // auth handlers
	dreamHandlers()     // dream handlers
	feedHandlers()      // user's feed handlers
	subscribeHandlers() // users' subscribe handlers
	likesHandlers()     // likes input handlers
}

func pingHandlers() {
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "msg": "pong", "ts": time.Now().Unix()})
	})

	r.GET("/api/pong", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "msg": "ping", "ts": time.Now().Unix()})
	})
}

// connect to redis and mongodb
func dbConn() (rdb *redis.Client, mdb *mongo.Client) {
	ctx := context.TODO()
	// connect to redis
	rdb = redis.NewClient(&redis.Options{
		Addr: viper.GetString("redis"),
		DB:   0, // use default DB
	})

	// redis test ping
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	// connect to mongodb
	mdb, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(viper.GetString("mongo")))
	if err != nil {
		panic(err)
	}

	// mongodb test ping
	if err = mdb.Ping(ctx, nil); err != nil {
		panic(err)
	}

	// setup mongo db
	db = mdb.Database(viper.GetString("db"))
	users = db.Collection(viper.GetString("users"))
	dreams = db.Collection(viper.GetString("dreams"))
	comments = db.Collection(viper.GetString("comments"))
	ensureIndeces()

	return
}

// find out the config file
func Config() {
	// os.Setenv("VP_MODE", "debug")
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.SetDefault("mode", "debug")
	viper.SetDefault("apiAddr", "localhost:5174")

	viper.SetDefault("mongo", "mongodb://localhost:27017")
	viper.SetDefault("db", "DreamWalker")
	viper.SetDefault("users", "users")
	viper.SetDefault("dreams", "dreams")
	viper.SetDefault("comments", "comments")

	viper.SetDefault("redis", "localhost:6379")

	viper.SetDefault("pwdMinStr", 50) // password minimal strengh, 40-70 maybe reasonable

	viper.SetDefault("expDream", time.Hour*1)         // dream details cache will expires in ONE hour by default
	viper.SetDefault("expOutbox", time.Hour*1)        // user's outbox cache will expires in ONE hour by default
	viper.SetDefault("expUser", time.Hour*1)          // user's cache will expires in ONE hour by default
	viper.SetDefault("expFeedUpdatedAt", time.Hour*1) // user's feed updated time cache will expires in ONE hour by default

	viper.SetDefault("expNewFeed", time.Minute*1) // user's new feed  cache will expires in 1 minute by default

	viper.SetDefault("feedLimit", 16)   // max feed length
	viper.SetDefault("outboxLimit", 24) // max outbox length

	viper.SetDefault("feedUpdatedLimit", -3) // default feed updated limit at 3 days ago

	// env vars must prefix with "vp",
	// eg: "VP_HELLO=12" in .env file, then viper.Get("hello")
	viper.SetEnvPrefix("vp")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
