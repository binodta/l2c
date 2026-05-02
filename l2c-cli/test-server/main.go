package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		fmt.Fprintf(w, "<h1>l2c-proxy Test Server</h1>")
		fmt.Fprintf(w, "<h3>It works!</h3>")
		fmt.Fprintf(w, "<p>Time: %s</p>", time.Now().Format(time.RFC1123))
		fmt.Fprintf(w, "<p>Method: %s</p>", r.Method)
		fmt.Fprintf(w, "<p>Path: %s</p>", r.URL.Path)
		fmt.Fprintf(w, "<h3>Headers:</h3><pre>")
		for k, v := range r.Header {
			fmt.Fprintf(w, "%s: %v\n", k, v)
		}
		fmt.Fprintf(w, "</pre>")
	})

	fmt.Println("Test server starting on http://localhost:8000")
	if err := http.ListenAndServe(":8000", mux); err != nil {
		log.Fatal(err)
	}
}
