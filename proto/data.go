package proto

import "encoding/json"

const (
	Heartbeat = 0
	Unicast   = 1
	Broadcast = 2
)

// Data 应用层协议
type Data struct {
	Op           int
	FromUserId   uint32
	FromUserName string
	ToUserId     uint32
	Msg          string
}

func (d *Data) Pack(p *Package) error {
	var err error
	p.Version = VersionContent
	p.Msg, err = json.Marshal(d)
	p.Length = p.GetPackageLength()
	return err
}

func (d *Data) Unpack(p *Package) error {
	var err error
	err = json.Unmarshal(p.Msg, &d)
	return err
}
