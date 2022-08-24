package server

import (
	"lab/internal/config"
	"lab/internal/redis"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WSConn struct {
	conn        *websocket.Conn
	closeChan   chan struct{}
	receiveChan chan []byte
	sendChan    chan []byte
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
		receiveChan: make(chan []byte, 1024),
		sendChan:    make(chan []byte, 1024),
	}

	close(wsConn.closeChan)
}

func Start() {
	http.HandleFunc("/", rootHandler)
	err := http.ListenAndServe("localhost:7960", nil)
	if err != nil {
		log.Println(err)
	}
}
