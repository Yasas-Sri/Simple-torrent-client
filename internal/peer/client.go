package peer

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
)

type PeerConnection struct {
	Conn     net.Conn
	Choked   bool
	Bitfield Bitfield
	PeerID   [20]byte
	InfoHash [20]byte
}

type PieceProgress struct {
	Index      int
	Buffer     []byte
	Downloaded int
	Requested  int
}

func NewPieceProgress(index int, length int) *PieceProgress {
	return &PieceProgress{
		Index:  index,
		Buffer: make([]byte, length),
	}
}

func (p *PieceProgress) CheckHash(expected [20]byte) error {
	hash := sha1.Sum(p.Buffer)
	if hash != expected {
		return fmt.Errorf("piece %d failed hash check", p.Index)
	}
	return nil
}

func (pc *PeerConnection) SendRequest(index, begin, length int) error {
	reqPayload := FormatRequest(index, begin, length)
	msg := &Message{ID: MsgRequest, Payload: reqPayload}
	_, err := pc.Conn.Write(msg.Serialize())
	return err
}

func (pc *PeerConnection) SendInterested() error {
	msg := &Message{ID: MsgInterested}
	_, err := pc.Conn.Write(msg.Serialize())
	return err
}

// AddBlock parses the MsgPiece payload and copies the data into the buffer
func (p *PieceProgress) AddBlock(payload []byte) error {
	if len(payload) < 8 {
		return fmt.Errorf("payload too short")
	}
	// First 4 bytes: index, Second 4 bytes: begin (offset)
	begin := int(binary.BigEndian.Uint32(payload[4:8]))
	data := payload[8:]

	if begin+len(data) > len(p.Buffer) {
		return fmt.Errorf("data out of bounds")
	}

	copy(p.Buffer[begin:], data)
	p.Downloaded += len(data)
	return nil
}
