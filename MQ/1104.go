package main

import (
	"github.com/redis/go-redis/v9"
	"fmt"
	"time"
	"context"
)

var ctx = context.Background()
var rdb *redis.Client

// 连接redis
func init() {
	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
}

// 生产者
func Producer(queueKey string) {
	for i := 1; i < 5; i++{
		msg := fmt.Sprintf("msg-%d", i)
		rdb.LPush(ctx, queueKey, msg)
		fmt.Printf("发送：%v\n", msg)
		time.Sleep(200 * time.Millisecond)
	}
}

// 消费者
func Consumer(queueKey string) {
	for{
		//0表示永久阻塞
		res, _ := rdb.BRPop(ctx, 0, queueKey).Result()
		fmt.Printf("收到消息：%v\n", res[1])
	}
}

func main() {
	go Consumer("demo")
	Producer("demo")
	select{}
}