package main
import(
	"html/template"
	"fmt"
	"net/http"
)

type User struct {
	Name string
	Age int
}

func SayHello(w http.ResponseWriter,r *http.Request) {
	tmpl,err := template.ParseFiles("./hello.html")
	if err != nil {
		fmt.Println("creat template failed,err = ",err)
	}
	users := []User{}
	tmpl.Execute(w,users)
}

func main() {
	http.HandleFunc("/",SayHello)
	err := http.ListenAndServe(":9090",nil)
	if err != nil {
		fmt.Println("creat Server failed err =",err)
	}	
}