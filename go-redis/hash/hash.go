package main

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	rdb.HSet(ctx, "user:name", "name", "adel", "age", 25)
	rdb.HIncrBy(ctx, "user:name", "age", 1)
	age, _ := rdb.HGet(ctx, "user:name", "age").Result()
	fmt.Println(age)
}
