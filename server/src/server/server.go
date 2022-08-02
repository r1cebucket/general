package server

import (
	"log"
	"math/rand"
	"net"
	"time"

	"tcpserver/packet"

	pd "tcpserver/proto"

	proto "google.golang.org/protobuf/proto"
)

type messageHandler func([]byte, *Client) error

type Server struct {
	addr     string
	listener net.TCPListener
	data     map[string]interface{}

	clientMap  map[string]Client
	msgChanMap map[string]chan interface{}

	handlers map[string]messageHandler
}

type User struct {
	Name   string `json:"name"`
	Passwd string `json:"passwd"`
}

type Client struct {
	Username string
	login    bool
	conn     net.Conn

	sendChan chan []byte
	quitChan chan bool
}

type Poem struct {
	Strains    []string `json:"strains"`
	Author     string   `json:"author"`
	Paragraphs []string `json:"paragraphs"`
	Title      string   `json:"title"`
}

type Author struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// client todo
func (s Server) stopClient(c Client) {
	c.conn.Close()
	c.login = false
	s.clientMap[c.Username] = c
	// log.Println("client stop", c)
}

func Start(port string) {
	server := Server{}

	// setup
	server.setupServer(port)

	// put into channel
	go server.logTicker()
	go server.acceptConn()
	// get out of channel
	go server.chanTrigger()
}

func (server *Server) setupServer(port string) {
	tcpAddr := "127.0.0.1:" + port
	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		log.Panic(err)
	}

	tcpListener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Panic(err)
	}

	server.addr = tcpAddr
	server.listener = *tcpListener
	server.clientMap = map[string]Client{}
	server.msgChanMap = map[string]chan interface{}{}
	server.data = ReadData()

	// add msg type
	server.msgChanMap["LogTicker"] = make(chan interface{}, 10)
	server.msgChanMap["AcceptConn"] = make(chan interface{}, 10)
	server.msgChanMap["Login"] = make(chan interface{}, 10)

	// add packet handlers
	server.handlers = map[string]messageHandler{}
	server.handlers["AuthRequest"] = server.authReqHandler
	server.handlers["Heartbeat"] = server.heartbeatHandler
	server.handlers["PoemResponse"] = server.poemResHandler
	server.handlers["BiographyRequest"] = server.BiogReqHandler

	log.Println("Server start at: " + server.addr)
}

func (server Server) logTicker() {
	ticker := time.NewTicker(5 * time.Second)

	for range ticker.C {
		server.msgChanMap["LogTicker"] <- nil
	}
	ticker.Stop()
}

func (server Server) acceptConn() {
	for {
		conn, err := server.listener.AcceptTCP()
		if err != nil {
			continue
		}
		server.msgChanMap["AcceptConn"] <- conn
	}
}

func (server *Server) chanTrigger() {
	// wait until there is message in the channel
	for {
		select {
		case <-server.msgChanMap["LogTicker"]:
			server.logConnNum()
		case connIf := <-server.msgChanMap["AcceptConn"]:
			conn := connIf.(net.Conn)
			server.handleConn(conn)
		case clientIf := <-server.msgChanMap["Login"]:
			c := clientIf.(*Client)
			server.handleLogin(*c)
		}
	}
}

// create Client to save info
// receive and send packet in the conn
func (s *Server) handleConn(conn net.Conn) {
	// after authentication, if the user already has a connection, then replace it
	c := Client{
		conn:     conn,
		login:    false,
		sendChan: make(chan []byte),
		quitChan: make(chan bool),
	}

	// send and receive
	// need quit for these two functions
	// another connection set up for the same user
	// need to quit the old client
	go s.receiveFromClient(c)
	go s.sendToClient(c)
}

func (s Server) receiveFromClient(c Client) { // todo timeout
	for {
		// SetReadDeadline sets the deadline for future Read calls
		c.conn.SetReadDeadline(time.Now().Add(time.Second * 60))
		p := packet.Packet{}
		if err := p.ReadFromConn(c.conn); err != nil {
			log.Println("read from the connection error:", err)
			break
		}
		// get handler with packet name
		handler, handlerExist := s.handlers[p.PacketName]
		if !handlerExist {
			log.Println("packet name undefined")
			continue
		}
		if err := handler(p.Payload, &c); err != nil {
			log.Println(err)
			break
		}
	}
	s.stopClient(c)
}

func (s Server) sendToClient(c Client) error {
	for {
		select {
		case byteArr := <-c.sendChan:
			c.conn.SetWriteDeadline(time.Now().Add(time.Second * 60))
			n, err := c.conn.Write(byteArr)
			if err != nil || n != len(byteArr) {
				return err
			}
		case <-c.quitChan:
			s.stopClient(c)
			return nil
		}
	}
}

// message handlers
func (s Server) handleLogin(c Client) {
	clientOld, connExist := s.clientMap[c.Username]
	// add the conn
	// if the username exist in map, renew
	if connExist && c.login {
		s.stopClient(clientOld)
		log.Println("connection exist, reset")
		s.clientMap[c.Username] = c
	} else if !connExist && c.login {
		log.Println("new connection")
		s.clientMap[c.Username] = c
	} else if !c.login {
		s.stopClient(c)
		log.Println("authentication error")
		return
	}

	// start to post poems
	go s.postPoem(c)
}

// function
func (s Server) postPoem(c Client) {
	// log.Println("start to post poems to client: " + c.Username)
	ticker := time.NewTicker(time.Second * 10)
	poems := s.data["poems"].([]Poem)
	for range ticker.C {
		if !c.login {
			break
		}
		rand.Seed(time.Now().Unix())
		poem := poems[rand.Intn(len(poems))]

		req := &pd.PoemRequest{
			Title:      poem.Title,
			Author:     poem.Author,
			Strains:    poem.Strains,
			Paragraphs: poem.Paragraphs,
		}

		p := packet.Packet{}
		payload, err := proto.Marshal(req)
		if err != nil {
			log.Println(err)
			break
		}
		p.MakePacket("PoemRequest", payload)

		c.sendChan <- p.Pack()
	}
	ticker.Stop()
}
