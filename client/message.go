package client

import (
	"encoding/binary"
	"fmt"
	"io"
)

type messageID uint8

const (
	// MsgChoke chokes the receiver
	msgChoke messageID = 0
	// MsgUnchoke unchokes the receiver
	msgUnchoke messageID = 1
	// MsgInterested expresses interest in receiving data
	msgInterested messageID = 2
	// MsgNotInterested expresses disinterest in receiving data
	msgNotInterested messageID = 3
	// MsgHave alerts the receiver that the sender has downloaded a piece
	msgHave messageID = 4
	// MsgBitfield encodes which pieces that the sender has downloaded
	msgBitfield messageID = 5
	// MsgRequest requests a block of data from the receiver
	msgRequest messageID = 6
	// MsgPiece delivers a block of data to fulfill a request
	msgPiece messageID = 7
	// MsgCancel cancels a request
	msgCancel messageID = 8
)

// Message stores ID and payload of a message
type message struct {
	ID      messageID
	Payload []byte
}

// FormatRequest creates a REQUEST message
func formatRequest(index, begin, length int) *message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &message{ID: msgRequest, Payload: payload}
}

// FormatHave creates a HAVE message
func formatHave(index int) *message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return &message{ID: msgHave, Payload: payload}
}

// ParsePiece parses a PIECE message and copies its payload into a buffer
func parsePiece(index int, buf []byte, msg *message) (int, error) {
	if msg.ID != msgPiece {
		return 0, fmt.Errorf("Expected PIECE (ID %d), got ID %d", msgPiece, msg.ID)
	}
	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("Payload too short. %d < 8", len(msg.Payload))
	}
	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("Expected index %d, got %d", index, parsedIndex)
	}
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("Begin offset too high. %d >= %d", begin, len(buf))
	}
	data := msg.Payload[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("Data too long [%d] for offset %d with length %d", len(data), begin, len(buf))
	}
	copy(buf[begin:], data)
	return len(data), nil
}

// ParseHave parses a HAVE message
func parseHave(msg *message) (int, error) {
	if msg.ID != msgHave {
		return 0, fmt.Errorf("Expected HAVE (ID %d), got ID %d", msgHave, msg.ID)
	}
	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("Expected payload length 4, got length %d", len(msg.Payload))
	}
	index := int(binary.BigEndian.Uint32(msg.Payload))
	return index, nil
}

// Serialize serializes a message into a buffer of the form
// <length prefix><message ID><payload>
// Interprets `nil` as a keep-alive message
func (m *message) serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}
	length := uint32(len(m.Payload) + 1) // +1 for id
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

// Read parses a message from a stream. Returns `nil` on keep-alive message
func readMessage(r io.Reader) (*message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf)

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(r, messageBuf)
	if err != nil {
		return nil, err
	}

	m := message{
		ID:      messageID(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	return &m, nil
}

func (m *message) name() string {
	if m == nil {
		return "KeepAlive"
	}
	switch m.ID {
	case msgChoke:
		return "Choke"
	case msgUnchoke:
		return "Unchoke"
	case msgInterested:
		return "Interested"
	case msgNotInterested:
		return "NotInterested"
	case msgHave:
		return "Have"
	case msgBitfield:
		return "Bitfield"
	case msgRequest:
		return "Request"
	case msgPiece:
		return "Piece"
	case msgCancel:
		return "Cancel"
	default:
		return fmt.Sprintf("Unknown#%d", m.ID)
	}
}

func (m *message) string() string {
	if m == nil {
		return m.name()
	}
	return fmt.Sprintf("%s [%d]", m.name(), len(m.Payload))
}
