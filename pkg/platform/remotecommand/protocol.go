package remotecommand

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type MessageType byte

const (
	StdoutData  MessageType = 0
	StdoutClose MessageType = 1
	StderrData  MessageType = 2
	StderrClose MessageType = 3
	StdinData   MessageType = 4
	StdinClose  MessageType = 5
	ExitCode    MessageType = 6
)

type Message struct {
	messageType MessageType
	exitCode    int64
	data        io.Reader

	bytes []byte
}

func newDataMessage(messageType MessageType, data []byte) *Message {
	return &Message{
		messageType: messageType,
		bytes:       data,
	}
}

func newCloseMessage(messageType MessageType) *Message {
	return &Message{
		messageType: messageType,
	}
}

func NewExitCodeMessage(exitCode int) *Message {
	return &Message{
		messageType: ExitCode,
		bytes:       binary.AppendVarint([]byte{}, int64(exitCode)),
	}
}

func ParseMessage(reader io.Reader) (*Message, error) {
	buf := bufio.NewReader(reader)
	messageTypeInt, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	switch MessageType(messageTypeInt) {
	case StdoutClose, StderrClose, StdinClose:
		return &Message{
			messageType: MessageType(messageTypeInt),
		}, nil
	case StdoutData, StderrData, StdinData:
		return &Message{
			messageType: MessageType(messageTypeInt),
			data:        buf,
		}, nil
	case ExitCode:
		exitCode, err := binary.ReadVarint(buf)
		if err != nil {
			return nil, fmt.Errorf("read exit code: %w", err)
		}

		return &Message{
			messageType: ExitCode,
			exitCode:    exitCode,
		}, nil
	default:
		return nil, fmt.Errorf("unrecognized message type %b", messageTypeInt)
	}
}

func (m *Message) Bytes() []byte {
	return append([]byte{byte(m.messageType)}, m.bytes...)
}
