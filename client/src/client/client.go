package client

import (
	"hash/adler32"
	"log"
	"net"
	"time"

	pd "../proto"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	UserID string
	Passwd string
}

type Poem struct {
	Strains    []string `json:strains`
	Author     string   `json:author`
	Paragraphs []string `json:paragraphs`
	Title      string   `json:title`
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

func sendPackage(pkg Pkg, conn net.Conn) error {
	byteArr := pkg.Pack()

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

	pkg := Pkg{0, uint8(len(name)), name, payloadBytes, 0}
	pkg.PkgLen = uint32(1 + len(name) + len(payloadBytes) + 4)
	pkg.Checksum = adler32.Checksum(pkg.Pack())
	sendPackage(pkg, conn)

	// wait for response
	mode := "big"
	pkgLenByte := make([]byte, 4)
	_, err = conn.Read(pkgLenByte)
	if err != nil {
		log.Println(err)
	}

	pkgLen := Decode(pkgLenByte, mode)

	byteArr := make([]byte, pkgLen+4)
	copy(byteArr[:4], pkgLenByte[:])
	conn.Read(byteArr[4:])

	pkg = Pkg{}
	pkg.Unpack(byteArr)

	checksum := pkg.Checksum
	pkg.Checksum = uint32(0)
	if checksum != adler32.Checksum(pkg.Pack()) {
		log.Println("checksum error")
	}

	if pkg.PkgName != "AuthResponse" {
		return false
	}

	res := &pd.AuthResponse{}
	err = proto.Unmarshal(pkg.Payload, res)
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

	pkg := Pkg{0, uint8(len(name)), name, payloadBytes, 0}
	pkg.PkgLen = uint32(1 + len(name) + len(payloadBytes) + 4)
	pkg.Checksum = adler32.Checksum(pkg.Pack())

	for {
		err := sendPackage(pkg, conn)
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
		pkgLenByte := make([]byte, 4)
		_, err := conn.Read(pkgLenByte)
		if err != nil {
			log.Println("connection closed")
			conn.Close()
			break
		}

		pkgLen := Decode(pkgLenByte, mode)

		byteArr := make([]byte, pkgLen+4)
		copy(byteArr[:4], pkgLenByte[:])
		conn.Read(byteArr[4:])

		pkg := Pkg{}
		pkg.Unpack(byteArr)

		handdlePkg(pkg, conn)
	}
}

func handdlePkg(pkg Pkg, conn net.Conn) {
	switch {
	case pkg.PkgName == "Heartbeat":
		{
			log.Println("heartbeat from server")
		}
	case pkg.PkgName == "PoemRequest":
		{
			req := &pd.PoemRequest{}
			err := proto.Unmarshal(pkg.Payload, req)
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
			resPkg := Pkg{}
			resPkg.makePkg("PoemResponse", payload)

			sendPackage(resPkg, conn)
		}
	case pkg.PkgName == "BiographyResponse":
		{
			res := &pd.BiographyResponse{}
			proto.Unmarshal(pkg.Payload, res)
			log.Println(res)
		}
	default:
		{
			log.Println("package name undefine:", pkg.PkgName)
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

	pkg := Pkg{}
	pkg.makePkg("BiographyRequest", payload)

	sendPackage(pkg, conn)
}
