package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"kafka-im/proto"
	"kafka-im/tools"
	"net/url"
	"os"
	"time"
)

type WsClient struct {
	userName string
	addr     string
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewWsClient(userName, addr string) *WsClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WsClient{
		userName: userName,
		addr:     addr,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *WsClient) Connect() {
	u := url.URL{Scheme: "ws", Host: c.addr, Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("websocket client error when dialing: %s\n", err.Error())
		return
	}
	defer conn.Close()
	fmt.Printf("connect server success\n")
	go c.write(conn)
	go c.read(conn)
	<-c.ctx.Done()
	fmt.Printf("disconnect server success\n")
}

func (c *WsClient) Disconnect() {
	c.cancel()
}

func (c *WsClient) write(conn interface{}) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Enter Message: ")
		if scanner.Scan() {
			input := scanner.Text() + "\n"
			var (
				data = new(proto.Data)
				err  error
			)
			if data, err = tools.ParseInput(c.userName, input); err != nil {
				fmt.Printf("websocket client error when packing(用户数据->应用层): %s\n", err.Error())
				return
			}
			conn.(*websocket.Conn).SetWriteDeadline(time.Now().Add(5 * time.Minute)) // 设置写操作超时时间
			dataStream, _ := json.Marshal(data)
			if err = conn.(*websocket.Conn).WriteMessage(websocket.TextMessage, dataStream); err != nil {
				fmt.Printf("websocket client error when writing: %s\n", err.Error())
				return
			}
		} else {
			fmt.Println("Error reading input:", scanner.Err())
			return
		}
	}
}

func (c *WsClient) read(conn interface{}) {
	conn.(*websocket.Conn).SetReadLimit(2048)
	conn.(*websocket.Conn).SetReadDeadline(time.Now().Add(5 * time.Minute))
	conn.(*websocket.Conn).SetPongHandler(func(string) error {
		conn.(*websocket.Conn).SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})
	for {
		_, message, err := conn.(*websocket.Conn).ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return
			}
		}
		if message == nil {
			return
		}
		msg := new(proto.Msg)
		if err = json.Unmarshal(message, msg); err != nil {
			return
		}
		fmt.Println(string(msg.Body))
	}
}
