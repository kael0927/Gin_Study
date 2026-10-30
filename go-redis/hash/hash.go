package main

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	rdb.HSet(ctx, "user:name", "adel", "age", 25, 0)
	rdb.Incr(ctx, "age")
	age, _ := rdb.HSet(ctx, "user:name", "age").Result()
	fmt.Println(age)
}
