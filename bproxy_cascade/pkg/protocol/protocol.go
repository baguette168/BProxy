package protocol

import (
	"encoding/binary"
	"io"

	"google.golang.org/protobuf/proto"
	pb "github.com/bproxy/bproxy/proto"
)

func WriteMessage(w io.Writer, msg *pb.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

func ReadMessage(r io.Reader) (*pb.Message, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	msg := &pb.Message{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, err
	}

	return msg, nil
}