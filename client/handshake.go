package client

import (
	"fmt"
	"io"
)

type handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

// New creates a new handshake with the standard pstr
func newHandshake(infoHash, peerID [20]byte) *handshake {
	return &handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

// Serialize serializes the handshake to a buffer
func (h *handshake) serialize() []byte {
	pstrlen := len(h.Pstr)
	bufLen := 49 + pstrlen
	buf := make([]byte, bufLen)
	buf[0] = byte(pstrlen)
	copy(buf[1:], h.Pstr)
	// Leave 8 reserved bytes
	copy(buf[1+pstrlen+8:], h.InfoHash[:])
	copy(buf[1+pstrlen+8+20:], h.PeerID[:])
	return buf
}

func readHandshake(r io.Reader) (*handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])

	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeBuf[pstrlen+8+20:])

	h := handshake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return &h, nil
}
