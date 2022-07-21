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
	"sync"
	"time"

	"../pkg"
	pd "../proto"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	ipAddr   string
	port     string
	listener net.TCPListener
	connMap  map[string]net.Conn
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

var connMutex sync.RWMutex

func Start(port string) {
	ipAddr := "127.0.0.1"
	data := ReadData()
	tcpAddr := ipAddr + ":" + port
	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		log.Panic(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Panic(err)
	}

	server := Server{
		ipAddr,
		port,
		*listener,
		map[string]net.Conn{},
	}

	log.Println("Server start at: " + server.port)

	connMap := map[string]net.Conn{}
	go server.getConnNum(connMap)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}

		go handdleConn(conn, connMap, data)
		// go readFromConn(conn)
		// go writeToConn(conn)
	}
}

// func trigger (){

// }

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
func handdleConn(conn net.Conn, connMap map[string]net.Conn, data map[string]interface{}) {
	for {
		// mode := "big"
		// pkgLenByte := make([]byte, 4)
		// _, err := conn.Read(pkgLenByte)
		// if err != nil {
		// 	log.Println("connection closed")
		// 	conn.Close()
		// 	deleteFromMap(connMap, conn)
		// 	return
		// }

		// pkgLen := Decode(pkgLenByte, mode)

		// byteArr := make([]byte, pkgLen+4)
		// copy(byteArr[:4], pkgLenByte[:])
		// conn.Read(byteArr[4:])

		// pkg := Pkg{}
		// pkg.Unpack(byteArr)

		p := pkg.Pkg{}
		p.ReadFromConn(conn)
		log.Println(p)

		handdlePkg(p, conn, connMap, data)
	}
}

func handdlePkg(p pkg.Pkg, conn net.Conn, connMap map[string]net.Conn, data map[string]interface{}) {
	// use table or map to save the connections (add lock)
	// if the user is in table, server can process the package with other package name
	// otherwise, the server can only process the AuthenRequest
	checksum := p.Checksum
	p.Checksum = uint32(0)
	if checksum != adler32.Checksum(p.Pack()) {
		log.Println("checksum error")
	}

	auth := true
	var sendErr error

	switch {
	case p.PkgName == "AuthRequest":
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

			resPkg := pkg.Pkg{}
			resPkg.MakePkg(name, payloadBytes)
			sendErr = sendPackage(resPkg, conn)

			if !auth || sendErr != nil {
				conn.Close()
				deleteFromMap(connMap, conn)
				return
			}

			go postPoem(conn, data["poems"].([]Poem))
		}
	case p.PkgName == "Heartbeat":
		{
			sendErr = sendPackage(p, conn)
		}
	case p.PkgName == "PoemResponse":
		{
			res := &pd.PoemResponse{}
			err := proto.Unmarshal(p.Payload, res)
			if err != nil {
				log.Println(err)
			}
			// log.Println("user received", res)
		}
	case p.PkgName == "BiographyRequest":
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

			resPkg := pkg.Pkg{}
			resPkg.MakePkg("BiographyResponse", payload)

			sendErr = sendPackage(resPkg, conn)
		}
	default:
		{
			log.Println("package name undefinde", p.PkgName)
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

		p := pkg.Pkg{}
		payload, err := proto.Marshal(req)
		if err != nil {
			log.Println(err)
		}
		p.MakePkg("PoemRequest", payload)

		err = sendPackage(p, conn)
		if err != nil {
			// conn.Close()
			return
		}

		time.Sleep(time.Minute * 10)
	}
}

func sendPackage(p pkg.Pkg, conn net.Conn) error {
	byteArr := p.Pack()

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
