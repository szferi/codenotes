package main

import (
	"fmt"
	"log"
	"os"

	"github.com/szferi/codenotes/go-template-engine/engine"
)

func main() {
	engine, err := engine.New(os.DirFS("templates"), "*.html")
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range engine.Layout().Templates() {
		fmt.Println(t.Name())
	}
	fmt.Println("--")
	err = engine.ExecuteTemplate(os.Stdout, "templates/index.html", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("--")
	err = engine.ExecuteTemplate(os.Stdout, "templates/page.html", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("--")
}
