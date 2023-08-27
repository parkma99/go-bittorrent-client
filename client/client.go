package client

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/parkma99/go-bittorrent-client/peers"
)

type bitfield []byte

func (bf bitfield) hasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	return bf[byteIndex]>>(7-offset)&1 != 0
}

// SetPiece sets a bit in the bitfield
func (bf bitfield) setPiece(index int) {
	byteIndex := index / 8
	offset := index % 8
	bf[byteIndex] |= 1 << (7 - offset)
}

// A Client is a TCP connection with a peer
type client struct {
	conn     net.Conn
	choked   bool
	bitfield bitfield
	peer     peers.Peer
	infoHash [20]byte
	peerID   [20]byte
}

func completeHandshake(conn net.Conn, infohash, peerID [20]byte) (*handshake, error) {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	req := newHandshake(infohash, peerID)
	_, err := conn.Write(req.serialize())
	if err != nil {
		return nil, err
	}

	res, err := readHandshake(conn)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(res.InfoHash[:], infohash[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", res.InfoHash, infohash)
	}
	return res, nil
}

func recvBitfield(conn net.Conn) (bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	msg, err := readMessage(conn)
	if err != nil {
		return nil, err
	}
	if msg.ID != msgBitfield {
		err := fmt.Errorf("expected bitfield but got ID %d", msg.ID)
		return nil, err
	}

	return msg.Payload, nil
}

// New connects with a peer, completes a handshake, and receives a handshake
// returns an err if any of those fail.
func newClient(peer peers.Peer, peerID, infoHash [20]byte) (*client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = completeHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bf, err := recvBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &client{
		conn:     conn,
		choked:   true,
		bitfield: bf,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}

// Read reads and consumes a message from the connection
func (c *client) read() (*message, error) {
	msg, err := readMessage(c.conn)
	return msg, err
}

// SendRequest sends a Request message to the peer
func (c *client) sendRequest(index, begin, length int) error {
	req := formatRequest(index, begin, length)
	_, err := c.conn.Write(req.serialize())
	return err
}

// SendInterested sends an Interested message to the peer
func (c *client) sendInterested() error {
	msg := message{ID: msgInterested}
	_, err := c.conn.Write(msg.serialize())
	return err
}

// SendNotInterested sends a NotInterested message to the peer
func (c *client) sendNotInterested() error {
	msg := message{ID: msgNotInterested}
	_, err := c.conn.Write(msg.serialize())
	return err
}

// SendUnchoke sends an Unchoke message to the peer
func (c *client) sendUnchoke() error {
	msg := message{ID: msgUnchoke}
	_, err := c.conn.Write(msg.serialize())
	return err
}

// SendHave sends a Have message to the peer
func (c *client) sendHave(index int) error {
	msg := formatHave(index)
	_, err := c.conn.Write(msg.serialize())
	return err
}
