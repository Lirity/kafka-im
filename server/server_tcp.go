package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"kafka-im/proto"
	"net"
	"sync"
	"time"
)

type TcpServer struct {
	addr  string           // ip:port
	lock  sync.RWMutex     // 读写锁保证map并发安全
	users map[uint32]*User // 管理用户长连接
	ch    chan *proto.Msg  // 广播channel
}

func NewTcpServer(addr string) *TcpServer {
	return &TcpServer{
		addr:  addr,
		users: make(map[uint32]*User),
		ch:    make(chan *proto.Msg, proto.ChannelBufferSize),
	}
}

func (s *TcpServer) Serve() {
	var (
		addr     *net.TCPAddr
		listener *net.TCPListener
		err      error
	)
	logrus.Infof("tcp server is starting ...")
	if addr, err = net.ResolveTCPAddr("tcp", s.addr); err != nil {
		logrus.Warnf("tcp server error when resolving: %s", err.Error())
		return
	}
	if listener, err = net.ListenTCP("tcp", addr); err != nil {
		logrus.Warnf("tcp server error when listening: %s", err.Error())
		return
	}
	defer listener.Close()
	logrus.Infof("tcp server is listening on %s", s.addr)
	for {
		var conn *net.TCPConn
		if conn, err = listener.AcceptTCP(); err != nil {
			logrus.Warnf("tcp server error when accepting: %s", err.Error())
		} else {
			logrus.Infof("tcp client connect success")
			user := s.register(conn)
			// 协程退出时需要关闭链接并删除用户
			go s.write(user)
			go s.read(user)
		}
	}
}

func (s *TcpServer) register(conn interface{}) *User {
	userId := uuid.New().ID()
	user := NewUser(userId)
	user.connTcp = conn.(*net.TCPConn)
	s.lock.Lock()
	s.users[userId] = user
	s.lock.Unlock()
	return user
}

func (s *TcpServer) deregister(user *User) {
	s.lock.Lock()
	delete(s.users, user.userId)
	s.lock.Unlock()
}

func (s *TcpServer) write(u *User) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		pack := new(proto.Package)
		select {
		case msg, ok := <-u.ch:
			if !ok {
				logrus.Warnf("tcp server err: the channel if user %d was closed", u.userId)
				return
			}
			if err := msg.Pack(pack); err != nil {
				logrus.Warnf("tcp server err when packing message(应用层->传输层): %s", err.Error())
				return
			}
			if err := pack.Pack(u.connTcp); err != nil {
				logrus.Warnf("tcp server err when packing message(传输层->链路层): %s", err.Error())
				return
			}
		case <-ticker.C:
			msg := new(proto.Msg)
			msg.Op = proto.Heartbeat
			if err := msg.Pack(pack); err != nil {
				logrus.Warnf("tcp server err when packing heartbeat(应用层->传输层): %s", err.Error())
				return
			}
			if err := pack.Pack(u.connTcp); err != nil {
				logrus.Warnf("tcp server err when packing heartbeat(传输层->链路层): %s", err.Error())
				return
			}
		}
	}
}

func (s *TcpServer) read(u *User) {
	defer func() {
		u.connTcp.Close()
		s.deregister(u)
	}()
	scanner := bufio.NewScanner(u.connTcp)
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
			logrus.Warnf("tcp server err: scan overtime\n")
			break
		}
		for scanner.Scan() {
			scannedPack := new(proto.Package)
			if err := scannedPack.Unpack(bytes.NewReader(scanner.Bytes())); err != nil {
				logrus.Warnf("tcp server err when unpacking(链路层->传输层): %s", err.Error())
			}
			data := new(proto.Data)
			if err := data.Unpack(scannedPack); err != nil {
				logrus.Warnf("tcp server err when unpacking(传输层->应用层): %s", err.Error())
				break
			}
			data.FromUserId = u.userId
			// 后续需要替换为写消息队列
			switch data.Op {
			case proto.Heartbeat:
			case proto.Unicast:
				s.users[data.ToUserId].ch <- &proto.Msg{
					Op:   proto.Unicast,
					Body: []byte(fmt.Sprintf("[私信] From %d: %s", u.userId, data.Msg)),
				}
			case proto.Broadcast:
				for _, user := range s.users {
					user.ch <- &proto.Msg{
						Op:   proto.Broadcast,
						Body: []byte(fmt.Sprintf("[公告] From %d: %s", u.userId, data.Msg)),
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			logrus.Warnf("tcp server get a err package: %s", err.Error())
			return
		}
	}
}
