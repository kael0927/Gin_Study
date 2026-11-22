package main
import "fmt"

type Payment interface {
	Pay(amount float64) string
}

type Alipay struct{}
func (a Alipay) Pay(amount float64) string{
	return fmt.Sprintf("支付宝支付 %.2f 元", amount)
} 

type Wechatpay struct{}
func (w *Wechatpay) Pay(amount float64) string{
	return fmt.Sprintf("微信支付 %.2f 元", amount)
}
// Checkout 只依赖 Payment 接口，不依赖具体支付方式
// 这就是“面向接口编程”的落点：结算逻辑和支付实现解耦
func Checkout(p Payment, amount float64) {
	fmt.Println("开始结算")
	fmt.Println(p.Pay(amount))
	fmt.Println("结算结束")
}
func main() {
	Checkout(Alipay{}, 78)

	Checkout(&Wechatpay{}, 90.12)
	// 接口值里保存了“动态类型 + 动态值”
	var p Payment = Alipay{}
	fmt.Printf("接口 p 的动态类型是：%T\n", p)
}