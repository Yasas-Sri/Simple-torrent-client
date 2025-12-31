package peer

import (
	"fmt"
	"io"
)

const (
	ProtocolStr    = "BitTorrent protocol"
	HandshakeSize  = 68
	PstrlenOffset  = 0
	ReservedOffset = 20
	InfoHashOffset = 28
	PeerIDOffset   = 48
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func NewHandshake(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		Pstr:     ProtocolStr,
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (h *Handshake) Serialize() []byte {
	buf := make([]byte, HandshakeSize)
	buf[PstrlenOffset] = byte(len(h.Pstr))

	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], make([]byte, 8))
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])

	return buf
}

func Read(r io.Reader) (*Handshake, error) {
	buf := make([]byte, HandshakeSize)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake: %w", err)
	}

	pstrlen := int(buf[PstrlenOffset])
	if pstrlen != 19 {
		return nil, fmt.Errorf("invalid protocol string length: %d", pstrlen)
	}

	res := &Handshake{
		Pstr:     string(buf[1 : pstrlen+1]),
		InfoHash: [20]byte(buf[InfoHashOffset:PeerIDOffset]),
		PeerID:   [20]byte(buf[PeerIDOffset:HandshakeSize]),
	}

	if res.Pstr != ProtocolStr {
		return nil, fmt.Errorf("invalid protocol string: %s", res.Pstr)
	}

	return res, nil
}
