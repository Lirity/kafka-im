package tools

import (
	"errors"
	"kafka-im/proto"
	"regexp"
	"strconv"
)

func ParseInput(userName, input string) (*proto.Data, error) {
	re := regexp.MustCompile(`\[(.*?)\]\s*(.*)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 3 {
		return nil, errors.New("invalid input format")
	}
	idStr, msg := matches[1], matches[2]
	var (
		id uint32
		op int
	)
	if idStr == "*" {
		op = proto.Broadcast
	} else {
		op = proto.Unicast
		id64, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return nil, errors.New("invalid user id")
		}
		id = uint32(id64)
	}
	data := &proto.Data{
		Op:           op,
		FromUserName: userName,
		ToUserId:     id,
		Msg:          msg,
	}
	return data, nil
}
