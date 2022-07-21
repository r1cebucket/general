package server

import (
	"hash/adler32"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	pd "../proto"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	ipAddr string
	port   string
}

type User struct {
	Name   string `json:"name"`
	Passwd string `json:"passwd"`
}

type Poem struct {
	Strains    []string `json:strains`
	Author     string   `json:author`
	Paragraphs []string `json:paragraphs`
	Title      string   `json:title`
}

type Author struct {
	Name string `json:name`
	Desc string `json:desc`
}

var connMutex sync.RWMutex

func Start(port string) {
	data := ReadData()
	server := Server{"127.0.0.1", port}
	listener := server.createListener()

	connMap := map[string]net.Conn{}
	go server.getConnNum(connMap)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}

		go handdleConn(conn, connMap, data)
	}
}

// log
func (s Server) getConnNum(connMap map[string]net.Conn) {
	for {

		// log_str := getPortConn(s.port)
		connMutex.RLock()
		log_str := "port " + string(s.port) + " TCP connection #: " + strconv.Itoa(len(connMap))
		connMutex.RUnlock()
		log.Println(log_str)
		log_file(log_str)
		time.Sleep(time.Second * 3)
	}
}

func getPortConn(port string) string {
	// netstat -nat|grep -i "8000"|wc -l
	output, err := exec.Command("/bin/sh", "-c", "netstat -nat|grep -i \""+string(port)+"\"|wc -l").Output()
	if err != nil {
		log.Panic(err)
	}
	log_str := "port " + string(port) + " TCP connection #:" + string(output)
	return log_str[:len(log_str)-1]
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
func (s Server) createListener() *net.TCPListener {
	tcpAddr := s.ipAddr + ":" + s.port
	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		log.Panic(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Panic(err)
	}
	log.Println("Server start at: " + s.port)
	return listener
}

func handdleConn(conn net.Conn, connMap map[string]net.Conn, data map[string]interface{}) {
	for {
		mode := "big"
		pkgLenByte := make([]byte, 4)
		_, err := conn.Read(pkgLenByte)
		if err != nil {
			log.Println("connection closed")
			conn.Close()
			deleteFromMap(connMap, conn)
			return
		}

		pkgLen := Decode(pkgLenByte, mode)

		byteArr := make([]byte, pkgLen+4)
		copy(byteArr[:4], pkgLenByte[:])
		conn.Read(byteArr[4:])

		pkg := Pkg{}
		pkg.Unpack(byteArr)

		handdlePkg(pkg, conn, connMap, data)
		time.Sleep(time.Millisecond * 50)
	}
}

func handdlePkg(pkg Pkg, conn net.Conn, connMap map[string]net.Conn, data map[string]interface{}) {
	// use table or map to save the connections (add lock)
	// if the user is in table, server can process the package with other package name
	// otherwise, the server can only process the AuthenRequest
	checksum := pkg.Checksum
	pkg.Checksum = uint32(0)
	if checksum != adler32.Checksum(pkg.Pack()) {
		log.Println("checksum error")
	}

	auth := true
	var sendErr error

	switch {
	case pkg.PkgName == "AuthRequest":
		{
			// authenticat the user name and passwd
			// add connection to map
			req := &pd.AuthRequest{}
			err := proto.Unmarshal(pkg.Payload, req)
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
			connMutex.Lock()
			connOld, connExist := connMap[req.Username]
			if pass {
				// add the conn
				// if the username exist in map, renew
				if connExist {
					connOld.Close()
					// log.Println("connection exist, renew")
					connMap[req.Username] = conn
				} else {
					// log.Println("new connection")
					connMap[req.Username] = conn
				}
			}
			connMutex.Unlock()

			resPkg := Pkg{}
			resPkg.makePkg(name, payloadBytes)
			sendErr = sendPackage(resPkg, conn)

			if !auth || sendErr != nil {
				conn.Close()
				deleteFromMap(connMap, conn)
				return
			}

			go postPoem(conn, data["poems"].([]Poem))
		}
	case pkg.PkgName == "Heartbeat":
		{
			// log.Println("heart beat package")
			// resPkg := Pkg{}
			// resPkg.makePkg("Heartbeat", make([]byte, 0))
			sendErr = sendPackage(pkg, conn)
		}
	case pkg.PkgName == "PoemResponse":
		{
			res := &pd.PoemResponse{}
			err := proto.Unmarshal(pkg.Payload, res)
			if err != nil {
				log.Println(err)
			}
			// log.Println("user received", res)
		}
	case pkg.PkgName == "BiographyRequest":
		{
			req := &pd.BiographyRequest{}
			err := proto.Unmarshal(pkg.Payload, req)
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

			resPkg := Pkg{}
			resPkg.makePkg("BiographyResponse", payload)

			sendErr = sendPackage(resPkg, conn)
		}
	default:
		{
			log.Println("package name undefinde", pkg.PkgName)
		}
	}

	if sendErr != nil {
		conn.Close()
		deleteFromMap(connMap, conn)
	}
}

func postPoem(conn net.Conn, poems []Poem) {
	for {
		rand.Seed(time.Now().Unix())
		poem := poems[rand.Intn(len(poems))]

		req := &pd.PoemRequest{
			Title:      poem.Title,
			Author:     poem.Author,
			Strains:    poem.Strains,
			Paragraphs: poem.Paragraphs,
		}

		pkg := Pkg{}
		payload, err := proto.Marshal(req)
		if err != nil {
			log.Println(err)
		}
		pkg.makePkg("PoemRequest", payload)

		err = sendPackage(pkg, conn)
		if err != nil {
			// conn.Close()
			return
		}

		time.Sleep(time.Minute * 10)
	}
}

func sendPackage(pkg Pkg, conn net.Conn) error {
	byteArr := pkg.Pack()

	_, err := conn.Write(byteArr)
	if err != nil {
		log.Println("send package err:", err)
		// conn.Close()
	}
	return err
}

func deleteFromMap(connMap map[string]net.Conn, conn net.Conn) {
	connMutex.Lock()
	for name := range connMap {
		if connMap[name] == conn {
			delete(connMap, name)
			connMutex.Unlock()
			return
		}
	}
}
