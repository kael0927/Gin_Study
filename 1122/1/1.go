/*定义 Payment 接口，两个实现 Alipay 和 WechatPay。写一个函数接收 Payment，用 type switch 分别打印不同支付方式的信息。

判断：接口里装的是 *SmsNotifier，用 n.(SmsNotifier) 能断言成功吗？为什么？

写一段代码：对 nil 接口做 type switch，让 case nil 分支命中。*/
package main
import "fmt"

type Payment interface {
	Pay(amount int)
}
type Alipay struct { name string }
func (a Alipay) Pay(amount int) {
	fmt.Println("alipay pay :", amount)
}
type WechatPay struct { name string}
func (w WechatPay) Pay(amount int) { 
	fmt.Println("wechatpay pay:", amount)
}
func main() {
	var p Payment = Alipay{ name: "alipay"}
	describe(p)
	p = WechatPay{ name: "wechatpay"}
	describe(p)
	var a Payment
	describe(a)
	
}
func describe(p Payment){
	switch v := p.(type){
	case Alipay:
		fmt.Println(v.name)
	case WechatPay:
		fmt.Println(v.name)
	case nil:
		fmt.Println("nil")
	default:
		fmt.Println("error")
	}
}

