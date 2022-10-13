package dream

import (
	"context"
	"encoding/json"
	"time"
)

func setCache(key string, d interface{}, exp time.Duration) error {
	p, err := json.Marshal(d)
	if err != nil {
		return err
	}
	err = rdb.Set(context.TODO(), key, p, exp).Err()
	return err
}

func getCache(key string, d interface{}) error {
	str, err := rdb.Get(context.TODO(), key).Result()
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(str), d)
	return err
}

func delCache(key string) error {
	return rdb.Del(context.TODO(), key).Err()
}

func expires(key string) error {
	err := rdb.ExpireAt(context.TODO(), key, time.Now()).Err()
	return err
}

func expiresIn(key string, duration time.Duration) error {
	return rdb.ExpireLT(context.TODO(), key, duration).Err()
	// err := rdb.ExpireAt(context.TODO(), key, time.Now()).Err()
	// return err
}
