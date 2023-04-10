package main

import (
	"fastun"
	"flag"
	"fmt"
)

var addr = flag.String("addr", ":80", "server listening address")
var cors = flag.String("cors", "*", "CORS header value")
var fallback = flag.String("fall", "", "fallback url")

var debug = flag.Bool("debug", false, "debug mode")

func main() {
	flag.Parse()
	if err := fastun.Serve(*addr, *cors, *fallback, *debug); err != nil {
		fmt.Println(err)
	}
}
