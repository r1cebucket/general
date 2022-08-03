package client

import (
	"log"
	"tcpclient/packet"
	pd "tcpclient/proto"

	proto "google.golang.org/protobuf/proto"
)

func (c *Client) authResHandler(payload []byte) error {
	resPayload := &pd.AuthResponse{}
	err := proto.Unmarshal(payload, resPayload)
	if err != nil {
		log.Println(err)
		return err
	}

	c.login = resPayload.Authorization

	if c.login {
		// log.Println("connection success as", c.user.UserID, ":", resPayload.Interpration)
	} else {
		log.Println("connection faild as", c.user.UserID, ":", resPayload.Interpration)
	}
	return nil
}

func (c Client) heartbeatHandler(payload []byte) error {
	// log.Println("heartbeat from server")
	return nil
}

func (c Client) poemReqHandler(payload []byte) error {
	req := &pd.PoemRequest{}
	err := proto.Unmarshal(payload, req)
	if err != nil {
		log.Println(err)
		return err
	}
	// log.Println(req)

	// response
	res := &pd.PoemResponse{
		Title: req.Title,
	}
	resPayload, err := proto.Marshal(res)
	if err != nil {
		log.Println(err)
		return err
	}
	resPacket := packet.Packet{}
	resPacket.MakePacket("PoemResponse", resPayload)

	c.sendChan <- resPacket.Pack()
	return nil
}

func (c Client) biogResHandler(paylaod []byte) error {
	res := &pd.BiographyResponse{}
	if err := proto.Unmarshal(paylaod, res); err != nil {
		log.Println(err)
		return err
	}
	log.Println(res)
	return nil
}
