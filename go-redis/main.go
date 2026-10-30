package main

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. 连接本地Redis
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	// 2.把 bob 存到 user：name 的 key 里
	rdb.Set(ctx, "user:name", "bob", 0)
	// 3. 取出user：name 的key
	name, _ := rdb.Get(ctx, "user:name").Result()
	fmt.Println("名字：", name)
}
