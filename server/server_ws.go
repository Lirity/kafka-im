package server

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"kafka-im/proto"
	"log"
	"net/http"
	"sync"
	"time"
)

type WsServer struct {
	addr  string           // ip:port
	lock  sync.RWMutex     // 读写锁保证map并发安全
	users map[uint32]*User // 管理用户长连接
	ch    chan *proto.Msg  // 广播channel
}

func NewWsServer(addr string) *WsServer {
	return &WsServer{
		addr:  addr,
		users: make(map[uint32]*User),
		ch:    make(chan *proto.Msg, proto.ChannelBufferSize),
	}
}

func (s *WsServer) Serve() {
	logrus.Infof("websocket server is starting ...")
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWs(w, r)
	})
	if err := http.ListenAndServe(s.addr, nil); err != nil {
		logrus.Warnf("websocket server error when listening: %s", err.Error())
		return
	}
	logrus.Infof("websocket server is listening on %s", s.addr)
}

func (s *WsServer) handleWs(w http.ResponseWriter, r *http.Request) {
	var upGrader = websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
	}
	//cross origin domain support
	upGrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, _ := upGrader.Upgrade(w, r, nil)
	logrus.Infof("websocket client connect success")
	user := s.register(conn)
	go s.write(user)
	go s.read(user)
}

func (s *WsServer) register(conn interface{}) *User {
	userId := uuid.New().ID()
	user := NewUser(userId)
	user.connWs = conn.(*websocket.Conn)
	s.lock.Lock()
	s.users[userId] = user
	s.lock.Unlock()
	return user
}

func (s *WsServer) deregister(user *User) {
	s.lock.Lock()
	delete(s.users, user.userId)
	s.lock.Unlock()
}

func (s *WsServer) write(u *User) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-u.ch:
			if !ok {
				logrus.Warnf("websocket server err: the channel if user %d was closed\n", u.userId)
				return
			}
			u.connWs.SetWriteDeadline(time.Now().Add(5 * time.Second)) // 设置写操作超时时间
			dataStream, _ := json.Marshal(msg)
			if err := u.connWs.WriteMessage(websocket.TextMessage, dataStream); err != nil {
				logrus.Warnf("websocket server when writing: %s", err.Error())
			}
		case <-ticker.C:
			msg := new(proto.Msg)
			msg.Op = proto.Heartbeat
			if err := u.connWs.WriteMessage(websocket.PingMessage, nil); err != nil {
				logrus.Warnf("websocket server when pinging: %s", err.Error())
				return
			}
		}
	}
}

func (s *WsServer) read(u *User) {
	defer func() {
		u.connTcp.Close()
		s.deregister(u)
	}()
	u.connWs.SetReadLimit(2048)
	u.connWs.SetReadDeadline(time.Now().Add(5 * time.Minute))
	u.connWs.SetPongHandler(func(string) error {
		u.connWs.SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})
	for {
		_, message, err := u.connWs.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Warnf("websocket server err: %s", err.Error())
				return
			}
		}
		if message == nil {
			log.Printf("websocket server err: %s", err.Error())
			return
		}
		data := new(proto.Data)
		if err = json.Unmarshal(message, data); err != nil {
			logrus.Warnf("websocket server err: %s", err.Error())
			return
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
}
