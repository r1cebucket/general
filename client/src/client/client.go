package client

import (
	"log"
	"net"
	"sync"
	"time"

	pd "tcpclient/proto"

	"tcpclient/packet"

	proto "google.golang.org/protobuf/proto"
)

type messageHandler func([]byte) error

type Client struct {
	user           User
	conn           net.Conn
	login          bool
	waitConn       sync.WaitGroup
	heartbeatTimer *time.Timer
	quit           bool
	localIP        string

	sendChan   chan []byte
	msgChanMap map[string]chan interface{}
	handlers   map[string]messageHandler
}

type User struct {
	UserID string
	Passwd string
}

type Poem struct {
	Strains    []string `json:"strains"`
	Author     string   `json:"author"`
	Paragraphs []string `json:"paragraphs"`
	Title      string   `json:"title"`
}

func (c *Client) Init(u User, localIP string) {
	c.user = u
	c.login = false
	c.sendChan = make(chan []byte, 10)
	c.waitConn.Add(1)
	c.heartbeatTimer = time.NewTimer(time.Second * 30)
	c.localIP = localIP
	c.quit = false

	c.msgChanMap = map[string]chan interface{}{}
	c.msgChanMap["Quit"] = make(chan interface{}, 1)
	c.msgChanMap["SetupConn"] = make(chan interface{}, 1)
	c.msgChanMap["SetupConn"] <- true

	// add headler
	c.handlers = map[string]messageHandler{}
	c.handlers["AuthResponse"] = c.authResHandler
	c.handlers["Heartbeat"] = c.heartbeatHandler
	c.handlers["PoemRequest"] = c.poemReqHandler
	c.handlers["BiographyResponse"] = c.biogResHandler
}

func (c *Client) Start() {
	go c.chanTrigger()

	go c.send()
	go c.receive()

	// go c.fetchDesc("宋太祖")
}

func (c *Client) chanTrigger() {
	for {
		select {
		case <-c.msgChanMap["SetupConn"]:
			c.connnetServer(c.localIP)
		case <-c.msgChanMap["Quit"]:
			log.Println("quit")
			c.stop()
			return
		case <-c.heartbeatTimer.C:
			c.sendHeartBeat()
		}

		if c.quit {
			break
		}
	}
}

func (c *Client) send() {
	for {
		if c.quit {
			break
		}
		c.waitConn.Wait()
		select {
		case byteArr := <-c.sendChan:
			c.conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
			n, err := c.conn.Write(byteArr)
			if err != nil || n != len(byteArr) {
				log.Println(err)
			}
			c.heartbeatTimer.Reset(time.Second * 30)
		}
	}
}

func (c *Client) receive() {
	for {
		if c.quit {
			break
		}
		c.waitConn.Wait()
		// log.Println("receiving...")
		p := packet.Packet{}
		if err := p.ReadFromConn(c.conn); err != nil {
			log.Println(err)
			c.waitConn.Add(1)
			log.Println("try to reconnect")
			c.msgChanMap["SetupConn"] <- true
			continue
		}
		handler := c.handlers[p.PacketName]
		if err := handler(p.Payload); err != nil {
			log.Println(err)
		}
	}
	// quit
}

func (c *Client) stop() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.login = false
	c.quit = true
}

func (c *Client) connnetServer(localIP string) {
	serverAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Println(err)
	}

	localAddr, err := net.ResolveTCPAddr("tcp", localIP+":"+"0")
	if err != nil {
		log.Println(err)
	}

	for i := 0; i < 3; i++ {
		c.conn, err = net.DialTCP("tcp", localAddr, serverAddr)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 5)
			continue
		} else {
			// log.Println("connect success")
			break
		}
	}

	if err != nil {
		c.msgChanMap["Quit"] <- true
		log.Println("try to connect to server faild")
		return
	}

	c.waitConn.Done()
	c.authenticate(c.conn)
}

func (c *Client) authenticate(conn net.Conn) {
	// make package
	name := "AuthRequest"
	reqPayload := &pd.AuthRequest{
		Username: c.user.UserID,
		Password: c.user.Passwd,
	}
	byteArr, err := proto.Marshal(reqPayload)
	if err != nil {
		log.Println(err)
	}
	req := packet.Packet{}
	req.MakePacket(name, byteArr)

	c.sendChan <- req.Pack()
}

func (c *Client) sendHeartBeat() {
	name := "Heartbeat"
	payload := make([]byte, 0)

	p := packet.Packet{}
	p.MakePacket(name, payload)

	c.sendChan <- p.Pack()
}

func (c *Client) fetchDesc(name string) {
	req := &pd.BiographyRequest{
		Name: name,
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		log.Println(err)
	}

	p := packet.Packet{}
	p.MakePacket("BiographyRequest", payload)

	c.sendChan <- p.Pack()
}
