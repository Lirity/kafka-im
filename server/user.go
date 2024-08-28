package server

import (
	"github.com/gorilla/websocket"
	"kafka-im/proto"
	"net"
)

type User struct {
	userId  uint32
	ch      chan *proto.Msg
	connTcp *net.TCPConn
	connWs  *websocket.Conn
}

func NewUser(userId uint32) *User {
	return &User{
		userId: userId,
		ch:     make(chan *proto.Msg, proto.ChannelBufferSize),
	}
}
