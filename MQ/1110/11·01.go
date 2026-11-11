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
	rdb *redis.Client
)

func initRedis() {
	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
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
		"x-dead-letter-exchange-key": dlqRouting,
	}
	if _, err := ch.QueueDeclare(bizQueue, true, false, false, false, bizArgs); err != nil {
		return err
	}
	//业务队列绑定
	if err := ch.QueueBind(bizQueue, bizRouting, bizExchange, false, nil); err != nil {
		return err
	}
	//声明DLQ 绑定DLX
	if _, err := ch.QueueDeclare(dlqQueue, true, false, false, false, nil); err != nil {
		return err
	}
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
			log.Printf("[Return]路由失败, msgID: %s, reason: %s", r.MessageId, r.ReplyText)
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
	for i := 0; i < 5; i++ {
		msgID:= fmt.Sprintf("msg-%d", i)
		pub := amqp091.Publishing{
			MessageId:	msgID,
			DeliveryMode:	amqp091.Persistent,
			Body:	[]byte(msgID),
		}
		if err := ch.Publish(bizExchange, bizRouting, true, false, pub); err != nil {
			log.Println("publish err", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Println("生产者发送完成")
}
//消费者（手动 ACK + QoS + 幂等 + 重试控制）
func consumer(ch *amqp091.Channel) {
	//QoS预取，一次只拿1条
	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatal("qos faile:",err)
	}
	deliveries, err := ch.Consume(bizQueue, "", false, false, false, false, nil);
	if err != nil {
		log.Fatal("consume faile:", err)
	}
	for d := range deliveries {
		msgID := d.MessageId
		log.Printf("[收到消息] msgID=%s, body=%s", msgID, d.Body)
		// ========== 幂等判断 start ==========
		ok, err := rdb.Exists(ctx, "mq_id:"+msgID).Result()
		if err != nil {
			log.Println("redis err,nack重试")
			_ = d.Nack(false, true)
			continue
		}
		if ok > 0 {
			log.Printf("[幂等跳过]%s 已经消费", msgID)
			_ = d.Ack(false)
			continue
		}
		// ========== 幂等判断 end  ===========
		// 模拟业务：msg-3 强制失败，用来测试死信
		if msgID == "msg-3" {
			log.Print("业务失败 第一次重试 第二次进入死信DLX")
			retryKey := "my_retry" + msgID
			cnt, _ := rdb.Incr(ctx, retryKey).Result()
			if cnt >= 2 {
				log.Print("重复次数用尽 进入死信")
				_ = d.Nack(false, false)// requeue=false → 进DLX
			} else {
				_ = d.Nack(false,true)// 放回队列重试
			}
			continue
		}
		// 正常业务成功
		log.Print("业务正常完成")
		// 标记已消费
		_, err = rdb.Set(ctx, "mq_id"+msgID, "1", 24*time.Hour).Result()
		_ = d.Ack(false)
	}
}
func main() {
	initRedis()
	conn, err := newRabbitConn()
	if err != nil {
		log.Fatal("connect rabbit fail:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("channel fail:", err)
	}
	defer ch.Close()

	if err := initTopology(ch); err != nil {
		log.Fatal("init topology fail:", err)
	}

	// 开协程跑消费者
	go consumer(ch)
	time.Sleep(1 * time.Second)

	// 生产者发消息
	producer(ch)

	<-make(chan struct{})
}

