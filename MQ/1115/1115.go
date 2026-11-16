package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

const (
	rabbitURL  = "amqp://guest:guest@127.0.0.1:5672/"
	exchange   = "biz.direct"
	queueName  = "biz.queue"
	routingKey = "biz.key"
)

type BizMsg struct {
	OrderId string
	MsgId   string
	Body    string
}

type Pending struct {
	dtag    uint64
	msg     amqp091.Publishing
	retries int
	sentAt  time.Time
	prev    []uint64 // 修复点2：重发链上的前驱 tag
}

type ReliableProducer struct {
	ch        *amqp091.Channel
	mu        sync.Mutex
	nextTag   uint64
	pending   map[uint64]*Pending
	maxRetry  int
	confirmCh chan amqp091.Confirmation
	dropTag   uint64 // 复现用：丢弃该 tag 的 ack；0=不丢
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewReliableProducer(ch *amqp091.Channel, maxRetry int, dropTag uint64) (*ReliableProducer, error) {
	if err := ch.Confirm(false); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &ReliableProducer{
		ch:        ch,
		pending:   make(map[uint64]*Pending),
		maxRetry:  maxRetry,
		confirmCh: ch.NotifyPublish(make(chan amqp091.Confirmation, 200)),
		dropTag:   dropTag,
		ctx:       ctx,
		cancel:    cancel,
	}
	p.wg.Add(2)
	go p.listenConfirm()
	go p.scanTimeout(500 * time.Millisecond)
	return p, nil
}

// 修复点1/4：所有 publish（含重发）唯一入口，锁内 nextTag++ 与 ch.Publish 严格同步
func (p *ReliableProducer) publishLocked(pub amqp091.Publishing, prev []uint64) (uint64, error) {
	p.nextTag++
	dtag := p.nextTag
	p.pending[dtag] = &Pending{dtag: dtag, msg: pub, sentAt: time.Now(), prev: prev}
	if err := p.ch.Publish(exchange, routingKey, false, false, pub); err != nil {
		delete(p.pending, dtag)
		return 0, err
	}
	return dtag, nil
}

func (p *ReliableProducer) Publish(biz BizMsg) error {
	pub := amqp091.Publishing{
		MessageId:    biz.MsgId,
		DeliveryMode: amqp091.Persistent,
		Body:         []byte(biz.Body),
	}
	p.mu.Lock()
	dtag, err := p.publishLocked(pub, nil)
	p.mu.Unlock()
	if err != nil {
		return err
	}
	fmt.Printf("[publish] dtag=%d msgId=%s\n", dtag, biz.MsgId)
	return nil
}

func (p *ReliableProducer) listenConfirm() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case conf, ok := <-p.confirmCh:
			if !ok {
				fmt.Println("[confirm] channel closed")
				return
			}
			// 复现：真的收到 ack，但生产者"看不见"
			if p.dropTag != 0 && conf.Ack && conf.DeliveryTag == p.dropTag {
				fmt.Printf("[producer] !!! 模拟丢弃 tag=%d 的 ack 回执\n", conf.DeliveryTag)
				continue
			}
			p.handleConfirm(conf)
		}
	}
}

func (p *ReliableProducer) handleConfirm(conf amqp091.Confirmation) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conf.Ack {
		// 直接清理单个 tag，不判断 Multiple
		p.cleanLocked(conf.DeliveryTag)
		fmt.Printf("[ack] single tag=%d，pending 剩 %d\n", conf.DeliveryTag, len(p.pending))
	} else {
		// Nack：同样只处理单个
		if item, exist := p.pending[conf.DeliveryTag]; exist {
			item.retries++
			item.sentAt = time.Now()
			fmt.Printf("[nack] tag=%d msgId=%s retries=%d\n", conf.DeliveryTag, item.msg.MessageId, item.retries)
		}
	}
}
// 修复点3：新 tag ack 时连带清理 prev 里的旧 tag
func (p *ReliableProducer) cleanLocked(tag uint64) {
	pm, ok := p.pending[tag]
	if !ok {
		return
	}
	for _, old := range pm.prev {
		if _, exists := p.pending[old]; exists {
			delete(p.pending, old)
			fmt.Printf("[producer] 连带清理旧 pending tag=%d\n", old)
		}
	}
	delete(p.pending, tag)
}

func (p *ReliableProducer) scanTimeout(timeout time.Duration) {
	defer p.wg.Done()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.scanOnce(timeout)
		}
	}
}

func (p *ReliableProducer) scanOnce(timeout time.Duration) {
	p.mu.Lock()
	var toResend []*Pending
	var exhausted []uint64
	for tag, pm := range p.pending {
		if time.Since(pm.sentAt) <= timeout {
			continue
		}
		if pm.retries < p.maxRetry {
			pm.retries++
			pm.sentAt = time.Now()
			toResend = append(toResend, pm)
		} else {
			exhausted = append(exhausted, tag) // 修复点5
		}
	}
	// 修复点5：耗尽即移除，避免每轮刷屏 + 泄漏
	for _, tag := range exhausted {
		if pm, ok := p.pending[tag]; ok {
			fmt.Printf("[dead] tag=%d msgId=%s 重试耗尽，丢弃/转死信\n", tag, pm.msg.MessageId)
			delete(p.pending, tag)
		}
	}
	p.mu.Unlock()

	for _, pm := range toResend {
		fmt.Printf("[retry] tag=%d msgId=%s 第 %d 次\n", pm.dtag, pm.msg.MessageId, pm.retries)
		prev := append([]uint64{}, pm.prev...)
		prev = append(prev, pm.dtag) // 修复点2：建立新旧关联
		p.mu.Lock()
		_, err := p.publishLocked(pm.msg, prev)
		p.mu.Unlock()
		if err != nil {
			fmt.Printf("[retry fail] tag=%d err=%v\n", pm.dtag, err)
		}
	}
}

func (p *ReliableProducer) Close() {
	p.cancel()
	p.wg.Wait() // 修复点7
}

// 消费者：手动 ack + 幂等
func startConsumer(wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		fmt.Println("consumer dial err:", err)
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		fmt.Println("consumer channel err:", err)
		return
	}
	defer ch.Close()
	if err := ch.Qos(10, 0, false); err != nil {
		fmt.Println("qos err:", err)
		return
	}
	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		fmt.Println("consume err:", err)
		return
	}

	processed := make(map[string]bool)
	var mu sync.Mutex
	for d := range deliveries {
		msgId := d.MessageId
		mu.Lock()
		exist := processed[msgId]
		mu.Unlock()
		if exist {
			fmt.Printf("[duplicate] msgId=%s 重复消息\n", msgId)
			_ = d.Ack(false)
			continue
		}

		// 修复点6：先处理，成功后再标记
		if err := doBusiness(d); err != nil {
			fmt.Printf("[consume] msgId=%s 业务失败: %v，重新入队\n", msgId, err)
			_ = d.Nack(false, true)
			continue
		}
		mu.Lock()
		processed[msgId] = true
		mu.Unlock()
		fmt.Printf("[consume] msgId=%s body=%s\n", msgId, string(d.Body))
		_ = d.Ack(false)
	}
}

func doBusiness(d amqp091.Delivery) error {
	time.Sleep(10 * time.Millisecond) // 模拟业务
	return nil
}

func main() {
	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	_ = ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		panic(err)
	}
	_ = ch.QueueBind(queueName, routingKey, exchange, false, nil)

	// dropTag=1：模拟第 1 条的 ack 丢失；改 0 则正常
	producer, err := NewReliableProducer(ch, 3, 1)
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go startConsumer(&wg)

	for i := 1; i <= 5; i++ {
		msg := BizMsg{
			OrderId: fmt.Sprintf("order-%d", i),
			MsgId:   fmt.Sprintf("msg-%d", i),
			Body:    fmt.Sprintf("data-%d", i),
		}
		_ = producer.Publish(msg)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(5 * time.Second)
	fmt.Println("exit")
}