package main

import (
	"fmt"
	"net/http"
	"os"
)

func sayHello(w http.ResponseWriter, r *http.Request) {
	b, _ := os.ReadFile("./hello.txt")
	_, _ = fmt.Fprintln(w, string(b))
}
func main() {
	http.HandleFunc("/hello", sayHello)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		fmt.Printf("http serve failed err:%v\n", err)
	}
}
