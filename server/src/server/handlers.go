package server

import (
	"log"
	"tcpserver/packet"
	pd "tcpserver/proto"

	proto "google.golang.org/protobuf/proto"
)

// packet handlers
func (s Server) authReqHandler(payload []byte, c *Client) error {
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

	c.sendChan <- resPacket.Pack()
	// save connection
	s.msgChanMap["Login"] <- c

	return nil
}

func (s Server) heartbeatHandler(payload []byte, c *Client) error {
	p := packet.Packet{}
	p.MakePacket("Heartbeat", payload)
	c.sendChan <- p.Pack()
	return nil
}

func (s Server) poemResHandler(payload []byte, c *Client) error {
	res := &pd.PoemResponse{}
	err := proto.Unmarshal(payload, res)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (s Server) BiogReqHandler(payload []byte, c *Client) error {
	if !c.login {
		return nil
	}
	req := &pd.BiographyRequest{}
	err := proto.Unmarshal(payload, req)
	if err != nil {
		log.Println(err)
	}
	// log.Println("client request description for:", req.Name)
	authors := s.data["authors"].(map[string]Author)

	res := &pd.BiographyResponse{
		Desc: authors[req.Name].Desc,
	}
	resPayload, err := proto.Marshal(res)
	if err != nil {
		log.Println(err)
	}

	resPacket := packet.Packet{}
	resPacket.MakePacket("BiographyResponse", resPayload)

	c.sendChan <- resPacket.Pack()
	return err
}
