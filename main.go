package main

import (
	"flag"
	"kafka-im/client"
)

func main() {
	var (
		net  = flag.String("net", "tcp", "client protocol")
		name = flag.String("name", "user", "client user name")
	)
	flag.Parse()
	var c client.Client
	if *net == "tcp" {
		c = client.NewTcpClient(*name, "localhost:8081")
	} else if *net == "ws" {
		c = client.NewWsClient(*userName, "localhost:8088")
	} else {
		panic("net err")
	}
	c.Connect()
}
