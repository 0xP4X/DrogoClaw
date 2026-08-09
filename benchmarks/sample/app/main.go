// Command vulnerableapp is a tiny, intentionally vulnerable web server used as
// a self-contained DrogonClaw benchmark challenge. It has a reflected SSTI
// endpoint that leaks the flag when its `name` parameter is evaluated.
//
// Build/run:
//
//	go run .            (listens on :8080)
//	go build -o vulnerableapp . && ./vulnerableapp
package main

import (
	"fmt"
	"net/http"
	"os"
)

const flag = "flag{DrogonClaw_SSTI_Bench_2026}"

func main() {
	addr := os.Getenv("BENCH_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		title := name
		if title == "" {
			title = "stranger"
		}
		// Vulnerable: the provided name is interpolated into a template-like
		// string and rendered. A SSTI payload such as
		//   {{.Flag}}
		// leaks the flag.
		body := fmt.Sprintf("<html><body><h1>Hello, %s!</h1></body></html>", title)
		if name == "{{.Flag}}" {
			body = fmt.Sprintf("<html><body><h1>Hello, %s!</h1></body></html>", flag)
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, body)
	})

	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "User-agent: *\nDisallow: /admin\n")
	})

	_ = http.ListenAndServe(addr, nil)
}
