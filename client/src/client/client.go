package client

import (
	"log"
	"net"
	"time"

	pd "tcpclient/proto"

	"tcpclient/packet"

	proto "google.golang.org/protobuf/proto"
)

type messageHandler func([]byte) error

type Client struct {
	user  User
	conn  net.Conn
	login bool

	sendChan   chan []byte
	msgChanMap map[string]chan interface{}
	handlers   map[string]messageHandler
	// heartbeatTimer time.Timer
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

func (c *Client) Init(u User) {
	c.user = u
	c.login = false
	c.sendChan = make(chan []byte, 10)

	c.msgChanMap = map[string]chan interface{}{}
	c.msgChanMap["Heartbeat"] = make(chan interface{}, 1)
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

func (c Client) Start() {
	// new design
	go c.send()
	go c.receive()

	go c.chanTrigger()

	go c.fetchDesc("宋太祖")
}

func (c Client) chanTrigger() {
	select {
	case <-c.msgChanMap["SetupConn"]:
		c.connnetServer("127.0.0.1", "8000")
	case <-c.msgChanMap["Heartbeat"]:
		c.sendHeartBeat()
	}

}

func (c Client) send() error {
	for {
		select {
		case byteArr := <-c.sendChan:
			c.conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
			n, err := c.conn.Write(byteArr)
			if err != nil || n != len(byteArr) {
				return err
			}

			// todo: update heartbeat timer
		case <-c.msgChanMap["Quit"]:
			c.stop()
			return nil
		}
	}
}

func (c Client) receive() {
	for {
		p := packet.Packet{}
		if err := p.ReadFromConn(c.conn); err != nil {
			log.Println(err)
			break
		}
		handler := c.handlers[p.PacketName]
		if err := handler(p.Payload); err != nil {
			log.Println(err)
		}
	}
	// quit
}

func (c *Client) stop() {
	c.conn.Close()
	c.login = false
}

func (c *Client) connnetServer(ipAddr string, port string) {
	serverAddr, err := net.ResolveTCPAddr("tcp", ipAddr+":"+port)
	if err != nil {
		log.Println(err)
	}

	for i := 0; i < 3; i++ {
		c.conn, err = net.DialTCP("tcp", nil, serverAddr)
		if err != nil {
			log.Println(err)
			continue
		} else {
			break
		}
	}
	if err != nil {
		c.msgChanMap["Quit"] <- true
		log.Println("try to connect to server faild")
		return
	}

	c.authenticate(c.conn)
	if !c.login {
		c.msgChanMap["Quit"] <- true
		return
	}
}

func (c Client) authenticate(conn net.Conn) {
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

func (c Client) fetchDesc(name string) {
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
