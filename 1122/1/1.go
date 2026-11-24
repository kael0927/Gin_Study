// 定义一个 Notifier 接口，包含 Send(msg string) error
// 实现 EmailNotifier 和 SmsNotifie
//  再写一个 NotifyUser(n Notifier) 函数调用它
package main
import "fmt"
type Notifier interface {
	Send(msg string) error
}
type EmailNotifier struct{ Name string }
func (e EmailNotifier) Send(msg string) error {
	fmt.Println("收到Email", msg)
	return nil
}


func NotifyUser(n Notifier, msg string) {
	err := n.Send(msg)
	if err != nil {
		fmt.Println("发送失败")
	}
}
func main() {
	var n Notifier
	fmt.Printf("n 的类型 %T   n == nil: %v\n", n, n==nil)
	n = EmailNotifier{ Name:"hui"}
	fmt.Printf("n 的类型 %T  n == nil: %v\n", n, n==nil)
	if e, ok := n.(EmailNotifier); ok {
		fmt.Println("断言成功，Name =", e.Name)
	}
}