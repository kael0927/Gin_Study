package main

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	
	score, _ := rdb.ZIncrBy(ctx, "hot:video", 1, "video:999").Result()
	fmt.Println(score)
}
