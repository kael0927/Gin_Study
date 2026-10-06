package main
import(
	"html/template"
	"fmt"
	"net/http"
)
func Kua(w http.ResponseWriter,r *http.Request) {
	kua := func(agr string) (string,error){
		return agr + "你好",nil
	}
	tmpl,err := template.New("index.html").
		Funcs(template.FuncMap{"kua":kua}).
		ParseFiles("./index.html","./header.html")
	if err != nil {
		fmt.Println("template.New err=",err)
	}
	data := map[string]interface{}{
    	"Name":  "小王子",
    	"Users": []string{"张三", "李四"},
	}
	tmpl.Execute(w,data)
}
func main() {
	http.HandleFunc("/",Kua)
	err := http.ListenAndServe(":9090",nil)
	if err != nil {
		fmt.Println("creat Server failed err =",err)
	}	
}