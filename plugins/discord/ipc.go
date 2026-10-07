package discord

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Discord IPC opcodes.
const (
	opHandshake = 0
	opFrame     = 1
	opClose     = 2
	opPing      = 3
	opPong      = 4
)

// maxFrame bounds a frame's JSON payload; Discord's messages are far smaller.
const maxFrame = 1 << 20

// frame is one IPC message: an opcode and a JSON payload.
type frame struct {
	op      uint32
	payload []byte
}

func writeFrame(w io.Writer, op uint32, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	message := make([]byte, 8, 8+len(payload))
	binary.LittleEndian.PutUint32(message, op)
	binary.LittleEndian.PutUint32(message[4:], uint32(len(payload)))
	_, err = w.Write(append(message, payload...))
	return err
}

func readFrame(r io.Reader) (frame, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return frame{}, err
	}
	f := frame{op: binary.LittleEndian.Uint32(header[:])}
	size := binary.LittleEndian.Uint32(header[4:])
	if size > maxFrame {
		return frame{}, fmt.Errorf("discord frame of %d bytes", size)
	}
	f.payload = make([]byte, size)
	if _, err := io.ReadFull(r, f.payload); err != nil {
		return frame{}, err
	}
	return f, nil
}

// message is an RPC command, response or event.
type message struct {
	Cmd   string          `json:"cmd"`
	Evt   string          `json:"evt,omitempty"`
	Nonce string          `json:"nonce,omitempty"`
	Args  any             `json:"args,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// rpcError is the data of an ERROR response or event.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e rpcError) Error() string {
	return fmt.Sprintf("discord error %d: %s", e.Code, e.Message)
}
