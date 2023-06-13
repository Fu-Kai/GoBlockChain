package main

import (
	"flag"
	"fmt"
	"log"
)

func init() {
	log.SetPrefix("Wallet Server: ")
}

func main() {
	port := flag.Uint("port", 8080, "TCP Port Number for Wallet Server")
	gateway := flag.String("gateway", "http://127.0.0.1:5000", "Blockchain Gateway")
	flag.Parse()
	fmt.Printf("port::%v gateway:%v\n", *port, *gateway)
	app := NewWalletServer(uint16(*port), *gateway)
	app.Run()
}
