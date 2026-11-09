package main

import (
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

const topicExchange = "demo_test"

// 生产者 往交换机法发消息
func producer(ch *amqp091.Channel) {
	routinKeys := []string{
		"user",
		"user.login",
		"user.login.sucess",
		"order.pay",
	}
	for _, rk := range routinKeys {
		msg := fmt.Sprintf("rk_%s", rk)
		ch.Publish(
			topicExchange,	//交换机类型
			rk, 	
			false,false,
			amqp091.Publishing{
				DeliveryMode: amqp091.Persistent,	//消息持久
				ContentType: "text/plain",
				Body: []byte(msg),
			},
		)
		fmt.Printf("发送消息：%s", msg)
		time.Sleep(500 * time.Millisecond)
	} 
}

func fanoutConsumer(ch *amqp091.Channel, name string, bindKey string) {
	// 声明交换机
	err := ch.ExchangeDeclare(
		topicExchange,
		"topic",
		true,
		false,false,false,nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	// 声明专属队列
	q, err := ch.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	err = ch.QueueBind(
		q.Name,
		bindKey,//路由键 无效
		topicExchange,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	// 监听自己的队列
	msgs, _ := ch.Consume(
		q.Name,
		"",
		true,false,false,false,nil,
	)
	fmt.Printf("[%s] 已订阅，绑定键：%s\n", name, bindKey)
	for d := range msgs {
		fmt.Printf("====> [%s] 收到消息：%s\n", name, d.Body)
	}
}

func main() {
	conn, _ := amqp091.Dial("amqp://guest:guest@127.0.0.1:5672/")
	defer conn.Close()

	ch1, _ := conn.Channel()
	ch2, _ := conn.Channel()
	pubCh, _ := conn.Channel()
	defer ch1.Close()
	defer ch2.Close()
	defer pubCh.Close()

	// 启动两个消费者，各有各的队列
	go fanoutConsumer(ch1, "订阅者A", "user.#")
	go fanoutConsumer(ch2, "订阅者B","user.*")

	time.Sleep(200 * time.Millisecond)
	producer(pubCh)

	fmt.Println("广播发送完成")
	select {}
}
