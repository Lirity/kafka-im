package proto

import "encoding/json"

const (
	ChannelBufferSize = 256
)

// Msg 为出消息队列后处理得到的消息 对应用户channel
type Msg struct {
	Op   int
	Body []byte
}

func (m *Msg) Pack(p *Package) error {
	var err error
	p.Version = VersionContent
	p.Msg, err = json.Marshal(m)
	p.Length = p.GetPackageLength()
	return err
}

func (m *Msg) Unpack(p *Package) error {
	var err error
	err = json.Unmarshal(p.Msg, &m)
	return err
}
