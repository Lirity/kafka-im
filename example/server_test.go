package example

import (
	"kafka-im/server"
	"testing"
)

func Test_TcpServer(t *testing.T) {
	s := server.NewTcpServer("localhost:8081")
	s.Serve()
}

func Test_WsServer(t *testing.T) {
	s := server.NewWsServer("localhost:8088")
	s.Serve()
}
