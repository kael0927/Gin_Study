package main

import (
	"fmt"
	"log"
	"time"
	"github.com/rabbitmq/amqp091-go"
)

func main() {
	// 和Rabbitmq建立TCP长连接
	conn, err := amqp091.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	// 程序结束关闭连接释放资源
	defer conn.Close()

	// 在连接上开一个通道
	ch, err := conn.Channel()
    if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	//创建一个队列
	queueName := "demo"
	_, err = ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	go consumer(ch, queueName)      // 异步启动消费者，一直监听
	time.Sleep(100 * time.Millisecond) // 等消费者先启动好
	producer(ch, queueName)        // 启动生产者发消息

	select {} // 阻塞住，不让程序退出，让消费者一直运行
}
//生产者函数 往指定队列里发消息
func producer(ch *amqp091.Channel, queueName string) {
	for i := 0; i < 5; i++ {
		body := fmt.Sprintf("msg-%d",i)
		
		err := ch.Publish(
			"",
			queueName,
			false,
			false,
			amqp091.Publishing{
				ContentType: "text/plain",//消息类型 纯文本
				Body: []byte(body), //消息内容必须转换成字节
			},
		)
		if err != nil {
			log.Printf("发送失败：%v", err)
		}
		fmt.Printf("发送msg-%s\n",body)
		time.Sleep(2*time.Second)
	}
}
//消费者函数 从队列里持续拿消息
func consumer(ch *amqp091.Channel, queueName string) {
	// 创建消费者 返回Go原生Channel 有新消息自动塞进来
	msg, err := ch.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	//循环读取
	go func() {
		for d := range msg {
			fmt.Printf("接收消息：%v",d.Body)
		}
	}()
}