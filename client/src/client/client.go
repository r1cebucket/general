package client

import (
	"hash/adler32"
	"log"
	"net"
	"time"

	pd "tcpclient/proto"

	"tcpclient/packet"

	proto "google.golang.org/protobuf/proto"
)

type Client struct {
	UserID string
	Passwd string
}

type Poem struct {
	Strains    []string `json:"strains"`
	Author     string   `json:"author"`
	Paragraphs []string `json:"paragraphs"`
	Title      string   `json:"title"`
}

func (client Client) Start() {
	conn := ConnnetServer("127.0.0.1", "8000")
	pass := client.Authenticate(conn)
	if !pass {
		return
	}
	go handdleConn(conn)
	go sendHeartBeat(conn)
	time.Sleep(time.Second * 30)
	go fetchDesc("宋太祖", conn)
}

func ConnnetServer(ipAddr string, port string) net.Conn {
	serverAddr, err := net.ResolveTCPAddr("tcp", ipAddr+":"+port)
	if err != nil {
		log.Println(err)
	}

	conn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		log.Println(err)
	}

	return conn
}

func sendPacket(p packet.Packet, conn net.Conn) error {
	byteArr := p.Pack()

	_, err := conn.Write(byteArr)
	if err != nil {
		log.Println("send package err:", err)
		conn.Close()
	}
	return err
}

func (c Client) Authenticate(conn net.Conn) bool {
	// make package
	name := "AuthRequest"
	payload := &pd.AuthRequest{
		Username: c.UserID,
		Password: c.Passwd,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Println(err)
	}

	p := packet.Packet{0, uint8(len(name)), name, payloadBytes, 0}
	p.PacketLen = uint32(1 + len(name) + len(payloadBytes) + 4)
	p.Checksum = adler32.Checksum(p.Pack())
	sendPacket(p, conn)

	// wait for response
	mode := "big"
	packetLenByte := make([]byte, 4)
	_, err = conn.Read(packetLenByte)
	if err != nil {
		log.Println(err)
	}

	packetLen := packet.Decode(packetLenByte, mode)

	byteArr := make([]byte, packetLen+4)
	copy(byteArr[:4], packetLenByte[:])
	conn.Read(byteArr[4:])

	p = packet.Packet{}
	p.Unpack(byteArr)

	checksum := p.Checksum
	p.Checksum = uint32(0)
	if checksum != adler32.Checksum(p.Pack()) {
		log.Println("checksum error")
	}

	if p.PacketName != "AuthResponse" {
		return false
	}

	res := &pd.AuthResponse{}
	err = proto.Unmarshal(p.Payload, res)
	if err != nil {
		log.Println(err)
	}

	if res.Authorization {
		log.Println("connection success as", c.UserID, ":", res.Interpration)
	} else {
		log.Println("connection faild as", c.UserID, ":", res.Interpration)
		conn.Close()
	}

	return res.Authorization
}

func sendHeartBeat(conn net.Conn) {
	name := "Heartbeat"
	payloadBytes := make([]byte, 0)

	p := packet.Packet{0, uint8(len(name)), name, payloadBytes, 0}
	p.PacketLen = uint32(1 + len(name) + len(payloadBytes) + 4)
	p.Checksum = adler32.Checksum(p.Pack())

	for {
		err := sendPacket(p, conn)
		if err != nil {
			log.Println("connection closed")
			conn.Close()
			break
		}
		time.Sleep(time.Second * 30)
	}
}

func handdleConn(conn net.Conn) {
	for {
		mode := "big"
		packetLenByte := make([]byte, 4)
		_, err := conn.Read(packetLenByte)
		if err != nil {
			log.Println("connection closed")
			conn.Close()
			break
		}

		packetLen := packet.Decode(packetLenByte, mode)

		byteArr := make([]byte, packetLen+4)
		copy(byteArr[:4], packetLenByte[:])
		conn.Read(byteArr[4:])

		p := packet.Packet{}
		p.Unpack(byteArr)

		handdlePacket(p, conn)
	}
}

func handdlePacket(p packet.Packet, conn net.Conn) {
	switch {
	case p.PacketName == "Heartbeat":
		{
			log.Println("heartbeat from server")
		}
	case p.PacketName == "PoemRequest":
		{
			req := &pd.PoemRequest{}
			err := proto.Unmarshal(p.Payload, req)
			if err != nil {
				log.Println(err)
			}
			log.Println(req)

			// response
			res := &pd.PoemResponse{
				Title: req.Title,
			}
			payload, err := proto.Marshal(res)
			if err != nil {
				log.Println(err)
			}
			resPacket := packet.Packet{}
			resPacket.MakePacket("PoemResponse", payload)

			sendPacket(resPacket, conn)
		}
	case p.PacketName == "BiographyResponse":
		{
			res := &pd.BiographyResponse{}
			proto.Unmarshal(p.Payload, res)
			log.Println(res)
		}
	default:
		{
			log.Println("package name undefine:", p.PacketName)
		}
	}
}

func fetchDesc(name string, conn net.Conn) {
	req := &pd.BiographyRequest{
		Name: name,
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		log.Println(err)
	}

	p := packet.Packet{}
	p.MakePacket("BiographyRequest", payload)

	sendPacket(p, conn)
}
