package server

import (
	"hash/adler32"
	"log"
)

type Pkg struct {
	PkgLen   uint32
	NameLen  uint8
	PkgName  string
	Payload  []byte
	Checksum uint32
}

func (pkg *Pkg) Unpack(byteArr []byte) {
	mode := "big"
	pkg.PkgLen = Decode(byteArr[:4], mode)

	pkg.NameLen = uint8(Decode(byteArr[4:5], mode))

	pkg.PkgName = string(byteArr[5 : 5+pkg.NameLen])
	pkg.Payload = byteArr[5+pkg.NameLen : len(byteArr)-4]

	pkg.Checksum = Decode(byteArr[pkg.PkgLen+4-4:], mode)
}

func (pkg *Pkg) Pack() []byte {
	mode := "big"
	byteArr := make([]byte, pkg.PkgLen+4)

	copy(byteArr[:4], Encode(pkg.PkgLen, 4, mode))
	copy(byteArr[4:5], Encode(uint32(pkg.NameLen), 1, mode))
	copy(byteArr[5:5+pkg.NameLen], []byte(pkg.PkgName))
	copy(byteArr[5+pkg.NameLen:pkg.PkgLen+4-4], pkg.Payload)

	copy(byteArr[pkg.PkgLen+4-4:], Encode(pkg.Checksum, 4, mode))
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

func (pkg *Pkg) makePkg(name string, payload []byte) {
	pkg.PkgLen = uint32(1 + len(name) + len(payload) + 4)
	pkg.NameLen = uint8(len(name))
	pkg.PkgName = name
	pkg.Payload = payload
	pkg.Checksum = 0
	pkg.Checksum = adler32.Checksum(pkg.Pack())
}
