package server

import (
	"log"
	"net"
	"tcpserver/packet"
	pd "tcpserver/proto"

	proto "google.golang.org/protobuf/proto"
)

type Client struct {
	Username string
	login    bool
	conn     net.Conn
	sendChan chan []byte
	quitChan chan bool
	handlers map[string]messageHandler
}

// packet handlers
func (c *Client) authReqHandler(payload []byte, s Server) error {
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

	log.Println(c)
	c.sendChan <- resPacket.Pack()
	// save connection
	s.msgChanMap["Login"] <- c

	return nil
}

func (c *Client) heartbeatHandler(payload []byte, s Server) error {
	p := packet.Packet{}
	p.MakePacket("Heartbeat", payload)
	c.sendChan <- p.Pack()
	log.Println(c)
	return nil
}

func (c *Client) poemResHandler(payload []byte, s Server) error {
	res := &pd.PoemResponse{}
	err := proto.Unmarshal(payload, res)
	if err != nil {
		log.Println(err)
	}
	return err
}
