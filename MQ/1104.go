package main

import (
	"github.com/redis/go-redis/v9"
	"fmt"
	"time"
	"context"
)

var ctx = context.Background()
var rdb *redis.Client
const (
	streamKey    = "stream_group_demo"
	groupName    = "order_group"
	consumerName = "consumer_1"
)
// 连接redis
func init() {
	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
}

// 生产者
func Publisher() {
	for i := 1; i < 5; i++{
		id,err := rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: streamKey,
			Values: map[string]interface{}{"data": fmt.Sprintf("msg-%d", i)},
		}).Result()
		if err == nil {
			fmt.Printf("发送成功, 消息ID:%s\n", id)
		}
		time.Sleep(2 * time.Second)
	}
}

// 消费者
func Subscriber() {
	lastID := "0"
	for{
		res, err := rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{streamKey, lastID},
			Count: 1,
			Block: 0,
		}).Result()
		if err != nil {
			continue
		}
		msg := res[0].Messages[0]
		fmt.Printf("收到消息 ID:%v, 内容: %v", msg.ID, msg.Values["data"])
		lastID = msg.ID
	}
}

func main() {
	go Subscriber()
	time.Sleep(1 * time.Second)
	Publisher()
	select{}
}