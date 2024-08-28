package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"kafka-im/proto"
	"kafka-im/tools"
	"net"
	"os"
)

type TcpClient struct {
	userName string
	addr     string
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewTcpClient(userName, addr string) *TcpClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &TcpClient{
		userName: userName,
		addr:     addr,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *TcpClient) Connect() {
	var (
		addr *net.TCPAddr
		err  error
	)
	if addr, err = net.ResolveTCPAddr("tcp", c.addr); err != nil {
		fmt.Printf("tcp client error when resolving: %s\n", err.Error())
		return
	}
	var conn *net.TCPConn
	if conn, err = net.DialTCP("tcp", nil, addr); err != nil {
		fmt.Printf("tcp client error when dialing: %s\n", err.Error())
		return
	}
	fmt.Printf("connect server success\n")
	go c.write(conn)
	go c.read(conn)
	<-c.ctx.Done()
	fmt.Printf("disconnect server success\n")
}

func (c *TcpClient) Disconnect() {
	c.cancel()
}

func (c *TcpClient) write(conn interface{}) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Enter Message: ")
		if scanner.Scan() {
			input := scanner.Text() + "\n"
			var (
				data = new(proto.Data)
				pack = new(proto.Package)
				err  error
			)
			if data, err = tools.ParseInput(c.userName, input); err != nil {
				fmt.Printf("tcp client error when packing(用户数据->应用层): %s\n", err.Error())
				return
			}
			if err = data.Pack(pack); err != nil {
				fmt.Printf("tcp client error when packing message(应用层->传输层): %s\n", err.Error())
				return
			}
			if err = pack.Pack(conn.(*net.TCPConn)); err != nil {
				fmt.Printf("tcp client error when packing message(传输层->链路层): %s\n", err.Error())
				return
			}
		} else {
			fmt.Println("Error reading input:", scanner.Err())
			return
		}
	}
}

func (c *TcpClient) read(conn interface{}) {
	scanner := bufio.NewScanner(conn.(*net.TCPConn))
	// 自定义Split函数用于切割TCP报文
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if !atEOF && data[0] == 'v' {
			if len(data) > int(proto.HeaderLength) {
				packLength := int16(0)
				_ = binary.Read(bytes.NewReader(data[proto.LengthStartIndex:proto.LengthEndIndex]), binary.BigEndian, &packLength)
				if int(packLength) <= len(data) { // 当读到的字节流大于等于定义的长度时返回数据
					return int(packLength), data[:packLength], nil
				}
			}
		}
		return
	})
	scanTimes := 0
	for {
		scanTimes++
		if scanTimes > 3 {
			fmt.Printf("tcp client scan overtime\n")
			break
		}
		for scanner.Scan() {
			scannedPack := new(proto.Package)
			if err := scannedPack.Unpack(bytes.NewReader(scanner.Bytes())); err != nil {
				fmt.Printf("tcp client error when unpacking(链路层->传输层): %s\n", err.Error())
			}
			msg := new(proto.Msg)
			if err := msg.Unpack(scannedPack); err != nil {
				fmt.Printf("tcp client error when unpacking(传输层->应用层): %s\n", err.Error())
				break
			}
			fmt.Println(string(msg.Body))
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("tcp server get a error package: %s\n", err.Error())
			return
		}
	}
}
