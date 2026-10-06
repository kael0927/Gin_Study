package main
import(
	"html/template"
	"fmt"
	"net/http"
)

type UserInfo struct {
	Name string
	Age int
}

func SayHello(w http.ResponseWriter,r *http.Request) {
	tmpl,err := template.ParseFiles("./hello.html")
	if err != nil {
		fmt.Println("creat template failed,err = ",err)
	}
	data := map[string]interface{}{
    "Title": "用户列表",
    "Users": []UserInfo{
        {Name: "张三", Age: 18},
        {Name: "李四", Age: 15},
        {Name: "王五", Age: 22},
    },
}
	tmpl.Execute(w,data)
}

func main() {
	http.HandleFunc("/",SayHello)
	err := http.ListenAndServe(":9090",nil)
	if err != nil {
		fmt.Println("creat Server failed err =",err)
	}	
}