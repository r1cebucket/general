package server

import (
	"encoding/json"
	"hash/adler32"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"tcpserver/packet"

	pd "tcpserver/proto"

	proto "google.golang.org/protobuf/proto"
)

type messageHandler func([]byte, Client) error

type Server struct {
	addr       string
	listener   net.TCPListener
	data       map[string]interface{}
	clientMap  map[string]Client
	msgChanMap map[string]chan interface{}
	handlers   map[string]messageHandler // byteArr -> handler -> byteArr in sendChan
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

type Client struct {
	Username string
	login    bool
	conn     net.Conn
	sendChan chan []byte
	quitChan chan bool
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

	// add handlers
	server.handlers = map[string]messageHandler{}
	server.handlers["AuthRequest"] = server.authReqHandler
	server.handlers["Heartbeat"] = server.heartbeatHandler

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
func (server *Server) handleConn(conn net.Conn) {
	// after authentication, if the user already has a connection, then replace it
	c := Client{}
	c.conn = conn
	c.login = false
	c.quitChan = make(chan bool)

	// send and receive
	// need quit for these two functions
	// another connection set up for the same user
	// need to quit the old client
	go server.receiveFromClient(c)
	go server.sendToClient(c)
}

func (server Server) receiveFromClient(c Client) { // todo timeout
	for {
		p := packet.Packet{}
		if err := p.ReadFromConn(c.conn); err != nil {
			log.Println("read from the connection error:", err)
			break
		}
		// get handler with packet name
		log.Println(p.PacketName)
		handler, handlerExist := server.handlers[p.PacketName]
		if !handlerExist {
			log.Println("packet name undefined")
			continue
		}
		if err := handler(p.Payload, c); err != nil {
			log.Println(err)
		}
	}
	c.stop()
}

func (server Server) sendToClient(c Client) error {
	for {
		select {
		case byteArr := <-c.sendChan:
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

// packet handlers
func (s Server) authReqHandler(payload []byte, c Client) error {
	// authenticat the user name and passwd
	// add connection to map
	req := &pd.AuthRequest{}
	if err := proto.Unmarshal(payload, req); err != nil {
		log.Println(err)
		return err
	}
	c.Username = req.Username

	var interp string
	users := s.data["users"].(map[string]User)
	if req.Password == users[req.Username].Passwd {
		c.login = true
		interp = "authentication pass"
	} else {
		c.login = false
		interp = "user name or password error"
	}
	log.Println(c.Username, interp)

	// send response
	name := "AuthResponse"

	resPayload := &pd.AuthResponse{
		Authorization: c.login,
		Interpration:  interp,
	}
	payloadBytes, err := proto.Marshal(resPayload)
	if err != nil {
		log.Panic(err)
	}

	resPacket := packet.Packet{}
	resPacket.MakePacket(name, payloadBytes)

	// sendErr := sendPacket(resPacket, c.conn)
	// if !c.login || sendErr != nil {
	// 	c.conn.Close()
	// 	// deleteFromMap(clientMap, conn)
	// 	return sendErr
	// }

	c.sendChan <- resPacket.Pack()
	// save connection
	s.msgChanMap["Login"] <- c

	return nil
}

func (s Server) heartbeatHandler(payload []byte, c Client) error {
	p := packet.Packet{}
	p.MakePacket("Heartbeat", payload)
	c.sendChan <- p.Pack()
	log.Println("heartbeat from client:" + c.Username)
	return nil
}

func 

// log
func (server Server) logConnNum() {
	// log_str := getPortConn(s.port)
	log_str := "port " + string(server.addr) + " TCP connection #: " + strconv.Itoa(len(server.clientMap))
	log.Println(log_str)
	log_file(log_str)
}

func log_file(s string) {
	file_name := "./conn_num.log"
	file, err := os.OpenFile(file_name, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		file, err = os.Create(file_name)
		if err != nil {
			log.Println(err)
		}
	}
	_, err = io.WriteString(file, s+"\n")
	if err != nil {
		log.Println(err)
	}
}

// function
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
				deleteFromMap(clientMap, conn)
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
		deleteFromMap(clientMap, conn)
	}
}

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
	}
	ticker.Stop()
}

func deleteFromMap(clientMap map[string]net.Conn, conn net.Conn) {
	// connMutex.Lock()
	for name := range clientMap {
		if clientMap[name] == conn {
			delete(clientMap, name)
			// connMutex.Unlock()
			return
		}
	}
}

func ReadData() map[string]interface{} {
	data := map[string]interface{}{}

	// read user info
	userInfoPath := "../data/userinfo/user.json"
	jsonByte := readFrom(userInfoPath)
	var jsonUser []map[string]string
	json.Unmarshal(jsonByte, &jsonUser)
	users := map[string]User{}
	for _, userMap := range jsonUser {
		user := User{userMap["name"], userMap["passwd"]}
		users[userMap["name"]] = user
	}
	data["users"] = users

	// read  poems
	root := "../data/poet/poem/"
	files, err := os.ReadDir(root)
	if err != nil {
		log.Println("open folder err:", err)
	}
	var jsonPoem []Poem
	poems := make([]Poem, 0)
	for _, file := range files {
		byteArr := readFrom(root + file.Name())
		json.Unmarshal(byteArr, &jsonPoem)
		poems = append(poems, jsonPoem...)
	}
	data["poems"] = poems

	//read authors
	root = "../data/poet/author/"
	files, err = os.ReadDir(root)
	if err != nil {
		log.Println("open folder err:", err)
	}
	authors := map[string]Author{}
	for _, file := range files {
		byteArr := readFrom(root + file.Name())
		var jsonAuthor []Author
		json.Unmarshal(byteArr, &jsonAuthor)
		for _, author := range jsonAuthor {
			authors[author.Name] = author
		}
	}
	data["authors"] = authors

	return data
}

func readFrom(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		log.Println("error opening file:", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Println("error reading file:", err)
	}

	return data
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
