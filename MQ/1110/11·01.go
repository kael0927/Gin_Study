package main 
import (
	"fmt"
	"time"
	"log"
	"context"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	// 业务交换机+队列
	bizExchange = "biz.direct"
	bizQueue = "biz.queue"
	bizRouting = "biz.key"
	// 死信DLX+DLQ
	dlxExchange = "dlx.Exchange"
	dlqQueue = "dlq.queue"
	dlqRouting = "dlq.key"

	rabbitRUL = "amqp://guest:guest@127.0.0.1:5672/"
	redisAddr = "127.0.0.1:6379"
)
var (
	ctx = context.Background()
	rbd *redis.Client
)

func initRedis() {
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal(err)
	}
}

func newRabbitConn() (*amqp091.Connection, error) {
	return amqp091.Dial(rabbitRUL)
}
// 初始化交换机、队列、绑定（DLX 配置挂在业务队列上）
func initTopology(ch *amqp091.Channel) error {
	// 声明业务交换机
	if err := ch.ExchangeDeclare(bizExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	// 声明DLX死信交换机
	if err := ch.ExchangeDeclare(dlxExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	//业务队列：durable 绑定DLX
	bizArgs := amqp091.Table{
		"x-dead-letter-exchange": dlxExchange,
		"x-dead-letter-exchange-queue": dlqQueue,
	}
	if _, err := ch.QueueDeclare(bizQueue, true, false, false, false, bizArgs); err != nil {
		return err
	}
	//业务队列绑定
	if err := ch.QueueBind(bizQueue, bizRouting, bizExchange, false, nil); err != nil {
		return err
	}
	//声明DLQ 绑定DLX
	if err := ch.QueueBind(dlqQueue, dlqRouting, dlxExchange, false, nil); err != nil {
		return err
	}
	return nil
}
// 生产者（Confirm + Return 回调）
func producer(ch *amqp091.Channel){
	//开启confirm模式
	if err := ch.Confirm(false); err != nil {
		log.Fatal(err)
	}
	confirmChan := ch.NotifyPublish(make(chan amqp091.Confirmation, 1))
	returnChan := ch.NotifyReturn(make(chan amqp091.Return, 1))
	// 监听return（路由失败退回）
	go func() {
		for r := range returnChan {
			log.Printf("[Return]路由失败, msgID: %s, reason: %s", r.MassageID, r.ReplyText)
		}
	}()
	// 监听confirm ack/nack
	go func() {
		for conf := range confirmChan {
			if conf.Ack {
				log.Printf("[Confirm ACK] deliveryTag:%d", conf.DeliveryTag)
			} else {
				log.Printf("[Confirm NACK] deliveryTag:%d", conf.DeliveryTag)
			}
		}
	}()
	// 发送消息，每条消息带唯一MessageId用于幂等
	
}