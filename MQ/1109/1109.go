package main 

import (
	"fmt"
	"log"
	"time"
	"github.com/rabbitmq/amqp091-go"
)

const directEx = "confirm_direct_ex"

func main() {
	conn, err := amqp091.Dial("amqp://guest:guest@127.0.0.1:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()
	//声明direct交换机
	err = ch.ExchangeDeclare(directEx, "direct", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	// 开启生产者确认
	err = ch.Confirm(false)
	if err != nil {
		log.Fatal(err)
	}
	//	接受ack/nack的通知channel
	notifyChan := ch.NotifyPublish(make(chan amqp091.Confirmation, 10))

	rk := "test_confirm"
	for i := 0; i <= 5; i++ {
		body := fmt.Sprintf("msg_%d", i)
		err := ch.Publish(
			directEx,
			rk,			
			false,
			false,
			amqp091.Publishing{
				DeliveryMode: amqp091.Persistent,
				Body: []byte(body),
			},
		)
		if err != nil {
			log.Printf("发送失败 %s, err=%v", body, err)
			continue
		}
		fmt.Printf("生产者已发出: %s\n", body)
	}
	go func() {
		for conf := range notifyChan {
			if conf.Ack {
				fmt.Printf("✅ 服务端确认收到消息, tag=%d\n", conf.DeliveryTag)
			} else {
				fmt.Printf("❌ 消息被nack, tag=%d，需要重试！\n", conf.DeliveryTag)
			}
		}
	}()

	time.Sleep(2 * time.Second)
	fmt.Println("发送完毕")
}