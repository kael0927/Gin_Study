package main

import (
	"context"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	// 延迟队列（放待超时消息，设置TTL）
	delayQueue     = "order.delay.queue"
	delayExchange  = "order.delay.dlx"
	targetQueue    = "order.check.queue"
	targetExchange = "order.biz.direct"
	routingKey     = "order.check.key"
	rabbitURL      = "amqp://guest:guest@127.0.0.1:5672/"
	redisAddr      = "127.0.0.1:6379"
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

// 初始化拓扑：完全沿用你原来的，不改动
func initOrderTopo(ch *amqp091.Channel) error {
	// 业务交换机
	if err := ch.ExchangeDeclare(targetExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	// 延迟队列死信交换机（保持direct，不改成fanout）
	if err := ch.ExchangeDeclare(delayExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	// 延迟队列：TTL=10s
	delayArgs := amqp091.Table{
		"x-dead-letter-exchange":    targetExchange,
		"x-dead-letter-routing-key": routingKey,
		"x-message-ttl": int32((30 * time.Second).Milliseconds()), // 单位ms
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

// ========== 【修改点】生产者函数，删掉confirm相关，只负责publish ==========
func orderProducer(ch *amqp091.Channel, orderId string) {
	msg := amqp091.Publishing{
		MessageId:    orderId,
		DeliveryMode: amqp091.Persistent,
		Body:         []byte(orderId),
	}
	if err := ch.Publish(delayExchange, "", true, false, msg); err != nil {
		log.Println("发送消息失败：", err)
	}
}

// 消费者：原样保留，只把吞掉的err补出来，不改动业务逻辑
func orderCheckConsumer(ch *amqp091.Channel) {
	if err := ch.Qos(1, 0, false); err != nil {
		log.Println("qos err:", err)
		return
	}
	deliveries, err := ch.Consume(targetQueue, "", false, false, false, false, nil)
	if err != nil {
		log.Println("consume err:", err)
		return
	}
	for d := range deliveries {
		orderId := string(d.Body)
		log.Printf("【超时检查】检查订单：%s", orderId)
		// 幂等判断：是否已经处理过
		exist, err := rdb.Exists(ctx, "order:checked:"+orderId).Result()
		if err != nil {
			log.Println("redis exists err:", err)
			if err := d.Nack(false, true); err != nil {
				log.Println("nack err:", err)
			}
			continue
		}
		if exist > 0 {
			log.Printf("订单%s已处理，跳过", orderId)
			if err := d.Ack(false); err != nil {
				log.Println("ack err:", err)
			}
			continue
		}

		// 模拟查订单状态：0=待支付 1=已支付
		status, err := rdb.Get(ctx, "order:"+orderId).Int()
		if err != nil {
			log.Println("get order status err:", err)
			if err := d.Nack(false, true); err != nil {
				log.Println("nack err:", err)
			}
			continue
		}
		if status == 1 {
			log.Printf("订单%s已付款，无需关闭", orderId)
		} else {
			log.Printf("订单%s超时未支付，执行关闭、释放库存！", orderId)
		}
		// 标记已处理
		if err := rdb.Set(ctx, "order:checked:"+orderId, "1", 24*time.Hour).Err(); err != nil {
			log.Println("set checked mark err:", err)
			if err := d.Nack(false, true); err != nil {
				log.Println("nack err:", err)
			}
			continue
		}
		if err := d.Ack(false); err != nil {
			log.Println("ack err:", err)
		}
	}
}

func main() {
	initRedis()
	conn, err := newConn()
	if err != nil {
		log.Fatal("conn fail:", err)
	}
	defer conn.Close()

	// ==========【修改点】拆分两个channel，生产者、消费者分开 ==========
	prodCh, err := conn.Channel()
	if err != nil {
		log.Fatal("prod channel fail:", err)
	}
	defer prodCh.Close()

	consCh, err := conn.Channel()
	if err != nil {
		log.Fatal("cons channel fail:", err)
	}
	defer consCh.Close()

	if err := initOrderTopo(prodCh); err != nil {
		log.Fatal("init topo err:", err)
	}

	// ==========【修改点】confirm只在这里初始化一次 ==========
	if err := prodCh.Confirm(false); err != nil {
		log.Fatal("confirm open err:", err)
	}
	confirmNotify := prodCh.NotifyPublish(make(chan amqp091.Confirmation, 10))
	go func() {
		for c := range confirmNotify {
			if c.Ack {
				log.Printf("投递确认，deliveryTag=%d", c.DeliveryTag)
			} else {
				log.Printf("投递NACK，deliveryTag=%d", c.DeliveryTag)
			}
		}
	}()


	// 可选：NotifyReturn，你想加就加，不想加可以注释掉
	returnChan := prodCh.NotifyReturn(make(chan amqp091.Return, 10))
	go func() {
		for r := range returnChan {
			log.Printf("消息退回 msgId:%s, reason:%s", r.MessageId, r.ReplyText)
		}
	}()

	// 启动消费者，使用独立channel
	go orderCheckConsumer(consCh)
	time.Sleep(1 * time.Second)

	// 模拟下单 order-001，状态待支付
	orderId := "order-001"
	rdb.Set(ctx, "order:"+orderId, 0, 0)
	orderProducer(prodCh, orderId)

	// 模拟下单 order-002，马上付款
	orderId2 := "order-002"
	rdb.Set(ctx, "order:"+orderId2, 0, 0)
	orderProducer(prodCh, orderId2)
	time.Sleep(2 * time.Second)
	rdb.Set(ctx, "order:"+orderId2, 1, 0)

	<-make(chan struct{})
}
