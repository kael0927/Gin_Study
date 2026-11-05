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
/*
## 第二部分：Stream 消费者组（生产级核心）
这是 Redis Stream 真正能用于业务队列的关键能力，解决「多消费者负载均衡 + 消息确认 + 失败重试」三大问题。
### 1. 核心概念
- 消费者组：绑定到一个 Stream 上，组内多个消费者共享消息，**一条消息只会分给组内一个消费者**，实现负载均衡
- 待处理列表（Pending）：消费者拿到消息后，消息先进入该消费者的待处理列表，不会删除
- ACK 确认（XACK）：消费者处理成功后执行 ACK，消息才从待处理列表移除
- 消息重试：消费者宕机后，它的待处理消息可以被转移给其他消费者重新处理
*/
func InitGroup() {
	err := rdb.XGroupCreateMkStream(ctx, streamKey, groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		panic(err)
	}
}
// 生产者
func producer() {
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
func groupConsumer() {
	for{
		res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: groupName,
			Consumer: consumerName,
			Streams: []string{streamKey, ">"},	// > 表示读取未分配给组内任何消费者的新消息
			Count: 1,
			Block: 0,
		}).Result()
		if err != nil {
			continue
		}
		msg := res[0].Messages[0]
		fmt.Printf("收到消息 ID:%v, 内容: %v", msg.ID, msg.Values["data"])
		time.Sleep(1*time.Second)
		rdb.XAck(ctx, streamKey, groupName, msg.ID)
	}
}

func main() {
	InitGroup()
	go groupConsumer()
	time.Sleep(1 * time.Second)
	producer()
	select{}
}