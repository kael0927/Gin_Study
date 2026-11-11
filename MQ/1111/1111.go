package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	// 延迟队列（放待超时消息，设置TTL）
	delayQueue    = "order.delay.queue"
	delayExchange = "order.delay.dlx"
	targetQueue   = "order.check.queue"
	targetExchange= "order.biz.direct"
	routingKey    = "order.check.key"
	rabbitURL     = "amqp://guest:guest@127.0.0.1:5672/"
	redisAddr     = "127.0.0.1:6379"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func initRedis() {
	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("redis connect err:", err)
	}
}

func newConn() (*amqp091.Connection, error) {
	return amqp091.Dial(rabbitURL)
}
// 初始化拓扑：延迟队列绑定DLX，消息TTL到期转发到业务交换机
func initOrderTopo(ch *amqp091.Channel) error {
	// 业务交换机
	if err := ch.ExchangeDeclare(targetExchange, "direct", true, false ,false, false, nil); err != nil {
		return err
	}
	// 延迟队列死信交换机
	if err := ch.ExchangeDeclare(delayExchange, "dirext", true, false, false, false, nil); err != nil {
		return err
	}
	// 延迟队列：TTL=30s（演示用，线上改成30*60=1800秒）
	delayArgs := amqp091.Table{
		"x-dead-letter-exchange":    targetExchange,
		"x-dead-letter-routing-key": routingKey,
		"x-message-ttl":             int32(10 * 1000), // 单位ms
	}
	if _, err := ch.QueueDeclare(delayQueue, true, false, false, false, delayArgs); err != nil {
		return err
	}
	if err := ch.QueueBind(delayQueue, "", delayExchange, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(targetQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(targetQueue, routingKey, targetExchange, false, nil); err != nil {
		return err
	}
	return nil
}
// 生产者：下单时发送延迟消息
func orderProducer(ch *amqp091.Channel, orderId string) {
	if err := ch.Confirm(false); err != nil {
		log.Fatal(err)
	}
	confirmNotify := ch.NotifyPublish(make(chan amqp091.Confirmation, 1))
	go func() {
		for c := range confirmNotify {
			if c.Ack {
				log.Printf("订单[%s]延迟消息投递成功", orderId)
			}
		}
	}()
	msg := amqp091.Publishing{
		
	}
}
