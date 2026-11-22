// 定义一个 Notifier 接口，包含 Send(msg string) error
// 实现 EmailNotifier 和 SmsNotifie
//  再写一个 NotifyUser(n Notifier) 函数调用它
package main
import "fmt"
type Notifier interface {
	Send(msg string) error
}
type EmailNotifier struct{}
func (e EmailNotifier) Send(msg string) error {
	fmt.Println("收到Email", msg)
	return nil
}
type SmsNotifier struct{}
func (s SmsNotifier) Send(msg string) error {
	fmt.Println("收到Sms", msg)
	return nil
}
func NotifyUser(n Notifier) {
	err := n.Send("lele")
	if err != nil {
		fmt.Println("发送失败")
	}
}
func main() {
	NotifyUser(EmailNotifier{})
	NotifyUser(SmsNotifier{})
}