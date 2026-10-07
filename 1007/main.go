package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("template/base.tmpl", "template/index.tmpl")

	if err != nil {
		fmt.Println("template.ParseGlob err = ", err)
		return
	}
	err = tmpl.ExecuteTemplate(w, "index.tmpl", nil)
	if err != nil {
		fmt.Println("渲染模板失败：", err)
		return
	}
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("template/base.tmpl", "template/about.tmpl")
	if err != nil {
		fmt.Println("模板解析失败：", err)
		return
	}
	err = tmpl.ExecuteTemplate(w, "about.tmpl", nil)
	if err != nil {
		fmt.Println("渲染模板失败：", err)
		return
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/about", aboutHandler)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		fmt.Println("服务器启动失败：", err)
	}
}
