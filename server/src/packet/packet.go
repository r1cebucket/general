package packet

import (
	"errors"
	"hash/adler32"
	"io"
	"log"
	"net"
)

type Packet struct {
	PacketLen  uint32
	NameLen    uint8
	PacketName string
	Payload    []byte
	Checksum   uint32
}

func (packet *Packet) Unpack(byteArr []byte) {
	mode := "big"
	packet.PacketLen = Decode(byteArr[:4], mode)

	packet.NameLen = uint8(Decode(byteArr[4:5], mode))

	packet.PacketName = string(byteArr[5 : 5+packet.NameLen])
	packet.Payload = byteArr[5+packet.NameLen : len(byteArr)-4]

	packet.Checksum = Decode(byteArr[packet.PacketLen+4-4:], mode)
}

func (packet *Packet) Pack() []byte {
	mode := "big"
	byteArr := make([]byte, packet.PacketLen+4)

	copy(byteArr[:4], Encode(packet.PacketLen, 4, mode))
	copy(byteArr[4:5], Encode(uint32(packet.NameLen), 1, mode))
	copy(byteArr[5:5+packet.NameLen], []byte(packet.PacketName))
	copy(byteArr[5+packet.NameLen:packet.PacketLen+4-4], packet.Payload)

	copy(byteArr[packet.PacketLen+4-4:], Encode(packet.Checksum, 4, mode))
	return byteArr
}

func Encode(num uint32, byteLen int, mode string) []byte {
	if byteLen > 4 || byteLen <= 0 {
		log.Panic("byte length out of range")
	}
	byteArr := make([]byte, byteLen)
	if mode == "big" {
		for i := 0; i < byteLen; i++ {
			byteArr[i] = byte(uint8(num >> ((byteLen - 1 - i) * 8)))
		}
	} else if mode == "little" {
		for i := 0; i < byteLen; i++ {
			byteArr[i] = byte(uint8(num >> (i * 8)))
		}
	}
	return byteArr
}

func Decode(byteArr []byte, mode string) uint32 {
	val := uint32(0)
	if mode == "big" {
		for i, b := range byteArr {
			val += uint32(b) << ((len(byteArr) - 1 - i) * 8)
		}
	} else if mode == "little" {
		for i, b := range byteArr {
			val += uint32(b) << (i * 8)
		}
	}

	return val
}

func (packet *Packet) MakePacket(name string, payload []byte) {
	packet.PacketLen = uint32(1 + len(name) + len(payload) + 4)
	packet.NameLen = uint8(len(name))
	packet.PacketName = name
	packet.Payload = payload
	packet.Checksum = 0
	packet.Checksum = adler32.Checksum(packet.Pack())
}

func (packet *Packet) ReadFromConn(conn net.Conn) error {
	mode := "big"
	PacketLenByte := make([]byte, 4)
	_, err := io.ReadFull(conn, PacketLenByte)
	if err != nil {
		log.Println("read package", err)
		return err
	}

	PacketLen := Decode(PacketLenByte, mode)

	byteArr := make([]byte, PacketLen+4)
	copy(byteArr[:4], PacketLenByte[:])
	n, err := io.ReadFull(conn, byteArr[4:])
	if err != nil || n != int(PacketLen) {
		return errors.New("read packet error")
	}

	packet.Unpack(byteArr)
	return nil
}
