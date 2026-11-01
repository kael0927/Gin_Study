package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	
	var fakerdb = map[string]string{
		"product1": "ipone15",
	}

	func GetProduct(ctx context.Context, rdb *redis.Client, id string) string {
		// 先查redis
		val, err := rdb.Get(ctx, id).Result()
		if err == nil {
			if val == "" {
				fmt.Println("命中空值缓存，防穿透成功")
				return ""
			}
			fmt.Println("缓存命中")
			return val
    	}
		if err == redis.Nil {
			fmt.Println("缓存未命中")
			dbval, exist := fakerdb[id]
			// 如果db中没有值
			if !exist {
				rdb.Set(ctx, id, "", 1*time.Minute)
				return ""
			}
			rdb.Set(ctx, id, dbval, 1*time.Minute)
			return dbval
		}

    }
}

