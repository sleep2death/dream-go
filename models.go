package dream

import (
	"context"
	"sort"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func addDream(d *dream) error {
	// insert dream into mongodb
	if _, err := dreams.InsertOne(context.TODO(), d); err != nil {
		return err
	}

	// cache dream with redis
	// if err := setCache("d:"+d.ID, d, viper.GetDuration("expDream")); err != nil {
	// 	return err
	// }

	// push the task into queue
	if err := rdb.RPush(context.TODO(), "DQ", d.ID).Err(); err != nil {
		return err
	}
	return nil
}

func updateDream(d *dream, keepCache bool) error {
	// insert dream into mongodb
	if _, err := dreams.UpdateByID(context.TODO(), d.ID, bson.M{"$set": d}); err != nil {
		return err
	}

	// if err := setCache("d:"+d.ID, d, viper.GetDuration("expDream")); err != nil {
	// 	return err
	// }

	// clear cache of the dream
	if !keepCache {
		expires("d:" + d.ID)
	} else {
		// or make the expire time much shorter
		expiresIn("d:"+d.ID, viper.GetDuration("expDreamShort"))
	}

	return nil
}

func getDreamById(id string) (d *dream, err error) {
	err = getCache("d:"+id, &d)
	// if the dream already cached
	if err == nil {
		l.Debugln("dream cached by redis:", id)
		return
	}

	// if error occcurs, ant it's not nil
	if err != nil && err != redis.Nil {
		return
	}

	// clear error
	err = nil

	// if err == redis.Nil, try to load if from mongo
	err = dreams.FindOne(context.TODO(), bson.M{"_id": id}, nil).Decode(&d)
	if err != nil {
		return
	}

	// cache the result dream
	err = setCache("d:"+d.ID, d, viper.GetDuration("expDream"))
	return
}

func addFeed(d *dream) error {
	_, err := users.UpdateOne(context.TODO(), bson.M{"username": d.Author}, bson.M{
		"$push": bson.M{
			"outbox": bson.M{
				"$each":     bson.A{&feed{Dream: d.ID, Generated: time.Now()}},
				"$position": 0,
				"$slice":    viper.GetInt("outboxLimit"),
			},
		}, "$set": bson.M{"updated": primitive.NewDateTimeFromTime(time.Now())},
	})

	// clear cache of the author
	expires("u:" + d.AuthorID)
	return err
}

// get user's outbox and cache it
func getUserById(id string, projections ...string) (usr user, err error) {
	err = getCache("u:"+id, &usr)

	if err == nil {
		l.Debugln("user cached by redis:", id)
		return
	}

	// if error occcurs, and it's not nil
	if err != nil && err != redis.Nil {
		return
	}

	// clear error
	err = nil

	l.Debugln("user not cached by redis:", id)

	match := bson.M{"_id": id}
	opts := options.FindOne()
	if len(projections) > 0 {
		proj := bson.M{}
		for _, p := range projections {
			proj[p] = 1
		}
		opts.SetProjection(proj)
	}

	err = users.FindOne(context.TODO(), match, opts).Decode(&usr)
	if err != nil {
		return
	}

	err = setCache("u:"+id, usr, viper.GetDuration("expUser"))
	return
}

func hasNewFeeds(id string, since time.Time) (hasNew []feed, err error) {
	usr, err := getUserById(id)
	if err != nil {
		return
	}

	err = getCache("u:"+id+":feed:new", &hasNew)

	if err != nil && err != redis.Nil {
		// l.Debugln("hasNew feed error:", err)
		return
	}

	// if cached
	if err == nil {
		l.Debugln("hasNew cached:", len(hasNew))
		return
	}

	// clear error
	err = nil

	// concat with self's outbox
	var list []feed = make([]feed, len(usr.Outbox))
	copy(list, usr.Outbox)

	// get subcription's outbox
	following := usr.Following
	for _, uid := range following {
		u, err := getUserById(uid, "outbox")
		if err != nil {
			return nil, err
		}
		list = append(list, u.Outbox...)
	}

	// filtered by since time
	// https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	bList := list[:0]
	for _, feed := range list {
		if feed.Generated.After(since) {
			// l.Debugln("append:", feed.Generated)
			bList = append(bList, feed)
		} else {
			l.Debugln("filtered out by generated time:", feed.Generated)
		}
	}

	if len(bList) == 0 {
		return
	}

	// filter with user's seen cache
	var seen []interface{}
	for _, f := range bList {
		seen = append(seen, f.Dream)
	}

	var flist []feed // filtered list
	iss, err := rdb.SMIsMember(context.TODO(), "u:"+id+":seen", seen...).Result()

	if err != nil && err != redis.Nil {
		return // if error
	} else if err == redis.Nil {
		flist = append(flist, bList...) // if cache not found
	} else {
		// filter with seen cache
		// l.Debugln("seen boolean", iss)
		for idx, is := range iss {
			if !is { // if not in seen cache
				flist = append(flist, bList[idx])
			}
		}
	}

	if len(flist) == 0 {
		return
	}

	// sort by generated time "desc"
	sort.Slice(flist, func(i, j int) bool {
		return flist[i].Generated.After(flist[j].Generated)
	})

	// cut the extra feeds
	if len(flist) > viper.GetInt("feedLimit") {
		flist = flist[:viper.GetInt("feedLimit")]
	}

	err = setCache("u:"+id+":feed:new", flist, viper.GetDuration("expNewFeed"))
	if err != nil {
		l.Debug("cache feed:new error:", err)
		return
	}

	return flist, nil
}

func getFeeds(id string, since time.Time) (feeds []*dream, err error) {
	// expires new feeds cache
	defer expires("u:" + id + ":feed:new")

	flist, err := hasNewFeeds(id, since)
	if err != nil || len(flist) == 0 {
		return
	}

	// get dream details
	for _, feed := range flist {
		d, err := getDreamById(feed.Dream)
		if err != nil {
			return feeds, err
		}
		feeds = append(feeds, d)
	}

	// cache seen list
	seen := make([]interface{}, len(flist))
	for idx, f := range flist {
		seen[idx] = f.Dream
	}

	res, err := rdb.SAdd(context.TODO(), "u:"+id+":seen", seen...).Result()
	l.Debugln("seen cache added number:", res)

	if err != nil {
		return
	}

	return
}

func addFollowing(uid string, following string) error {
	res, err := users.UpdateByID(context.TODO(), uid, bson.M{
		"$addToSet": bson.M{
			"following": following,
		}})
	l.Debugln("add following:", res.ModifiedCount)

	if err != nil {
		return err
	}

	res, err = users.UpdateByID(context.TODO(), following, bson.M{
		"$addToSet": bson.M{
			"followers": uid,
		}})
	l.Debugln("add follower:", res.ModifiedCount)

	if err != nil {
		return err
	}

	return err
}

func removeFollowing(uid string, following string) error {
	res, err := users.UpdateByID(context.TODO(), uid, bson.M{
		"$pull": bson.M{
			"following": following,
		}})
	l.Debugln("remove following:", res.ModifiedCount)

	if err != nil {
		return err
	}

	res, err = users.UpdateByID(context.TODO(), following, bson.M{
		"$pull": bson.M{
			"followers": uid,
		}})
	l.Debugln("remove follower:", res.ModifiedCount)

	if err != nil {
		return err
	}

	return err
}
