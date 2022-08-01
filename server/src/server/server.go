package server

import (
	"hash/adler32"
	"log"
	"math/rand"
	"net"
	"time"

	"tcpserver/packet"

	pd "tcpserver/proto"

	proto "google.golang.org/protobuf/proto"
)

type messageHandler func([]byte, Server) error

type Server struct {
	addr       string
	listener   net.TCPListener
	data       map[string]interface{}
	clientMap  map[string]Client
	msgChanMap map[string]chan interface{}
}

type User struct {
	Name   string `json:"name"`
	Passwd string `json:"passwd"`
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
func (c Client) stop() {
	c.conn.Close()
	return
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

	log.Println("Server start at: " + server.addr)
}

func (server Server) logTicker() {
	ticker := time.NewTicker(3 * time.Second)

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
			c := clientIf.(Client)
			server.handleLogin(c)
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

	c.handlers = map[string]messageHandler{}
	c.handlers["AuthRequest"] = c.authReqHandler
	c.handlers["Heartbeat"] = c.heartbeatHandler
	c.handlers["PoemResponse"] = c.poemResHandler

	// send and receive
	// need quit for these two functions
	// another connection set up for the same user
	// need to quit the old client
	go s.receiveFromClient(c)
	go s.sendToClient(c)
}

func (s Server) receiveFromClient(c Client) { // todo timeout
	for {
		p := packet.Packet{}
		if err := p.ReadFromConn(c.conn); err != nil {
			log.Println("read from the connection error:", err)
			break
		}
		// get handler with packet name
		handler, handlerExist := c.handlers[p.PacketName]
		if !handlerExist {
			log.Println("packet name undefined")
			continue
		}
		if err := handler(p.Payload, s); err != nil {
			log.Println(err)
		}
	}
	c.stop()
}

func (s Server) sendToClient(c Client) error {
	for {
		select {
		case byteArr := <-c.sendChan:
			log.Println("send packet")
			n, err := c.conn.Write(byteArr)
			if err != nil || n != len(byteArr) {
				return err
			}
		case <-c.quitChan:
			c.stop()
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
		clientOld.stop() // todo chan?
		log.Println("connection exist, reset")
		s.clientMap[c.Username] = c
	} else if !connExist && c.login {
		log.Println("new connection")
		s.clientMap[c.Username] = c
	} else if !c.login {
		c.stop()
		log.Println("authentication error")
	}

	// start to post poems
	go s.postPoem(c)
}

// function
func (s Server) postPoem(c Client) {
	ticker := time.NewTicker(time.Minute * 10)
	poems := s.data["poems"].([]Poem)
	for range ticker.C {
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
		}
		p.MakePacket("PoemRequest", payload)

		err = sendPacket(p, c.conn)
		if err != nil {
			break
		}
		log.Println("send poem to client")
	}
	ticker.Stop()
}

// Drop!!!!!!!!!!
func sendPacket(p packet.Packet, conn net.Conn) error {
	byteArr := p.Pack()
	_, err := conn.Write(byteArr)
	if err != nil {
		log.Println("send packet err:", err)
		// conn.Close()
	}
	return err
}

func handdlePacket(p packet.Packet, conn net.Conn, clientMap map[string]net.Conn, data map[string]interface{}) {
	// use table or map to save the connections (add lock)
	// if the user is in table, server can process the packet with other packet name
	// otherwise, the server can only process the AuthenRequest
	checksum := p.Checksum
	p.Checksum = uint32(0)
	if checksum != adler32.Checksum(p.Pack()) {
		log.Println("checksum error")
	}

	auth := true
	var sendErr error

	switch {
	case p.PacketName == "AuthRequest":
		{
			// authenticat the user name and passwd
			// add connection to map
			req := &pd.AuthRequest{}
			err := proto.Unmarshal(p.Payload, req)
			if err != nil {
				log.Println(err)
			}
			var pass bool
			var interp string
			users := data["users"].(map[string]User)
			if req.Password == users[req.Username].Passwd {
				pass = true
				interp = "authentication pass"
			} else {
				pass = false
				interp = "user name or password error"
			}
			auth = pass
			// log.Println(pass, interp)

			// send response
			name := "AuthResponse"

			payload := &pd.AuthResponse{
				Authorization: pass,
				Interpration:  interp,
			}
			payloadBytes, err := proto.Marshal(payload)
			if err != nil {
				log.Panic(err)
			}

			// save connection
			// connMutex.Lock()
			connOld, connExist := clientMap[req.Username]
			if pass {
				// add the conn
				// if the username exist in map, renew
				if connExist {
					connOld.Close()
					// log.Println("connection exist, renew")
					clientMap[req.Username] = conn
				} else {
					// log.Println("new connection")
					clientMap[req.Username] = conn
				}
			}
			// connMutex.Unlock()

			resPacket := packet.Packet{}
			resPacket.MakePacket(name, payloadBytes)
			sendErr = sendPacket(resPacket, conn)

			if !auth || sendErr != nil {
				conn.Close()
				// deleteFromMap(clientMap, conn)
				return
			}

			// go postPoem(conn, data["poems"].([]Poem))
		}
	case p.PacketName == "Heartbeat":
		{
			sendErr = sendPacket(p, conn)
		}
	case p.PacketName == "PoemResponse":
		{
			res := &pd.PoemResponse{}
			err := proto.Unmarshal(p.Payload, res)
			if err != nil {
				log.Println(err)
			}
			// log.Println("user received", res)
		}
	case p.PacketName == "BiographyRequest":
		{
			req := &pd.BiographyRequest{}
			err := proto.Unmarshal(p.Payload, req)
			if err != nil {
				log.Println(err)
			}
			// log.Println("client request description for:", req.Name)
			authors := data["authors"].(map[string]Author)

			res := &pd.BiographyResponse{
				Desc: authors[req.Name].Desc,
			}
			payload, err := proto.Marshal(res)
			if err != nil {
				log.Println(err)
			}

			resPacket := packet.Packet{}
			resPacket.MakePacket("BiographyResponse", payload)

			sendErr = sendPacket(resPacket, conn)
		}
	default:
		{
			log.Println("packet name undefinde", p.PacketName)
		}
	}

	if sendErr != nil {
		conn.Close()
		// deleteFromMap(clientMap, conn)
	}
}
