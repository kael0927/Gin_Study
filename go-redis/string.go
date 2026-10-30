package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. 连接本地Redis
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	// 2.把 bob 存到 user：name 的 key 里
	rdb.Set(ctx, "user:name", "bob", 10*time.Second)
	// 3. 取出user：name 的key
	name, _ := rdb.Get(ctx, "user:name").Result()
	fmt.Println("名字：", name)
	ttl, _ := rdb.TTL(ctx, "user:name").Result()
	fmt.Println("ttl:", ttl)
	rdb.Del(ctx, "user:name")
	name, err := rdb.Get(ctx,"user:123").Result()
	if err == redis.Nil {
		fmt.Println("缓存未命中")
	}else if err != nil {
		fmt.Println(err)
	}else{
		fmt.Println(name)
	}
}
