package server

import (
	"lab/internal/config"
	"lab/internal/redis"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WSConn struct {
	conn        *websocket.Conn
	closeChan   chan struct{}
	receiveChan chan Message
	sendChan    chan Message
	once        sync.Once
}

type Message struct {
	messageType int
	data        []byte
}

func init() {
	config.Init()
	redis.Init(config.Redis.Host, config.Redis.Port)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	wsConn := WSConn{
		conn:        conn,
		closeChan:   make(chan struct{}, 1),
		receiveChan: make(chan Message, 1024),
		sendChan:    make(chan Message, 1024),
		once:        sync.Once{},
	}

	go wsConn.receive()
	go wsConn.send()
	go wsConn.process()
}

func (wsConn WSConn) receive() {
	for {
		messageType, bytes, err := wsConn.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			wsConn.close()
			log.Println("receive stop")
			return
		}
		msg := Message{
			messageType: messageType,
			data:        bytes,
		}
		wsConn.receiveChan <- msg
	}
}

func (wsConn WSConn) send() {
	for {
		select {
		case msg := <-wsConn.sendChan:
			err := wsConn.conn.WriteMessage(msg.messageType, msg.data)
			if err != nil {
				log.Println(err)
			}
		case <-wsConn.closeChan:
			log.Println("send stop")
			return
		}
	}
}

func (wsConn WSConn) process() {

	for {
		select {
		case msg := <-wsConn.receiveChan:
			resRedis, err := redis.Do("get", "key")
			bytes, err := redis.String(resRedis, err)
			if err != nil {
				log.Println(err)
			}
			resMsg := Message{
				messageType: msg.messageType,
				data:        []byte(bytes),
			}
			wsConn.sendChan <- resMsg
		case <-wsConn.closeChan:
			log.Println("process stop")
			return
		}
	}
}

func (wsConn WSConn) close() {
	wsConn.once.Do(func() {
		wsConn.conn.Close()
		close(wsConn.closeChan)
	})
}

func Start() {
	http.HandleFunc("/", rootHandler)
	err := http.ListenAndServe("localhost:7960", nil)
	if err != nil {
		log.Println(err)
	}
}
