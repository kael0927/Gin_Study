package main

import (
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

const fanoutExchange = "hello_demo_test"

// 生产者 往交换机法发消息
func producer(ch *amqp091.Channel) {
	for i := 0; i <= 5; i++ {
		body := fmt.Sprintf("msg-%d",i)
		ch.Publish(
			fanoutExchange,	//交换机类型
			"", 	// fanout类型下路由键无效
			false,false,
			amqp091.Publishing{
				DeliveryMode: amqp091.Persistent,	//消息持久
				ContentType: "text/plain",
				Body: []byte(body),
			},
		)
		fmt.Printf("发送消息：%s", body)
		time.Sleep(500 * time.Millisecond)
	} 
}

func fanoutConsumer(ch *amqp091.Channel, name string) {
	// 声明交换机
	err := ch.ExchangeDeclare(
		fanoutExchange,
		"fanout",
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
		"",//路由键 无效
		fanoutExchange,
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
	for d := range msgs {
		fmt.Printf("[%s] 收到: %s\n", name, d.Body)
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
	go fanoutConsumer(ch1, "订阅者A")
	go fanoutConsumer(ch2, "订阅者B")

	time.Sleep(200 * time.Millisecond)
	producer(pubCh)

	fmt.Println("广播发送完成")
	select {}
}
