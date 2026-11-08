package main

import (
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

const queueName = "hello_demo"

// 生产者：循环发5条消息
func producer(ch *amqp091.Channel) {
	for i := 0; i < 5; i++ {
		body := fmt.Sprintf("msg-%d", i)
		err := ch.Publish(
			"",        // 默认交换机
			queueName, // 路由键=队列名
			false,
			false,
			amqp091.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			},
		)
		if err != nil {
			log.Printf("发送失败: %v", err)
		}
		fmt.Printf("发送: %s\n", body)
		time.Sleep(500 * time.Millisecond)
	}
}

// 消费者：持续监听队列
func consumer(ch *amqp091.Channel, name string) {
	msgs, err := ch.Consume(
		queueName,
		"",
		true, // autoAck 自动确认
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for d := range msgs {
			fmt.Printf("[%s]消费: %s\n", name, d.Body)
		}
	}()
}

func main() {
	// 1. 建立连接
	conn, err := amqp091.Dial("amqp://guest:guest@127.0.0.1:5672/")
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()

	// 2. 开启通道
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("开启通道失败:", err)
	}
	defer ch.Close()

	// 3. 声明队列
	_, err = ch.QueueDeclare(
		queueName,
		true, // 不持久化
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("声明队列失败:", err)
	}

	// 4. 启动消费者和生产者
	go consumer(ch, "1")
	go consumer(ch, "2")
	time.Sleep(100 * time.Millisecond)
	producer(ch)

	fmt.Println("消息发送完成，消费者持续运行中...")
	select {}
}
