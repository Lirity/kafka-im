package proto

import (
	"encoding/binary"
	"io"
)

const (
	VersionLength    int16 = 2
	LengthLength     int16 = 2
	HeaderLength           = VersionLength + LengthLength
	LengthStartIndex       = VersionLength
	LengthEndIndex         = HeaderLength
)

var (
	VersionContent = [2]byte{'v', '1'}
)

// Package 传输层协议，封装了数据长度解决粘包
type Package struct {
	Version [2]byte
	Length  int16
	Msg     []byte
}

func (p *Package) Pack(writer io.Writer) error {
	var err error
	err = binary.Write(writer, binary.BigEndian, &p.Version)
	err = binary.Write(writer, binary.BigEndian, &p.Length)
	err = binary.Write(writer, binary.BigEndian, &p.Msg)
	return err
}

func (p *Package) Unpack(reader io.Reader) error {
	var err error
	err = binary.Read(reader, binary.BigEndian, &p.Version)
	err = binary.Read(reader, binary.BigEndian, &p.Length)
	p.Msg = make([]byte, p.Length-HeaderLength)
	err = binary.Read(reader, binary.BigEndian, &p.Msg)
	return err
}

func (p *Package) GetPackageLength() int16 {
	length := HeaderLength + int16(len(p.Msg))
	return length
}
