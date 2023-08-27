package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/parkma99/go-bittorrent-client/bencode"
	"github.com/parkma99/go-bittorrent-client/client"
)

// Port to listen on
const Port uint16 = 65534

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
	Files       []fileInfo
}

type fileInfo struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type bencodeInfo struct {
	Length      int        `bencode:"length"`
	Files       []fileInfo `bencode:"files"`
	Name        string     `bencode:"name"`
	PieceLength int        `bencode:"piece length"`
	Pieces      string     `bencode:"pieces"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()
	o, _, err := bencode.Bdecode(file)
	if err != nil {
		return TorrentFile{}, err
	}
	dir, err := o.Dict()
	if err != nil {
		return TorrentFile{}, err
	}
	info_obj, exist := dir["info"]
	if !exist {
		return TorrentFile{}, errors.New("torrent file do not have info")
	}
	info_bytes := info_obj.Raw()
	bto := bencodeTorrent{}
	err = bencode.Unmarshal(o, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	return bto.toTorrentFile(info_bytes)
}

func (t *TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	copy(peerID[:], "-qB3150-123456789000")
	peers, err := t.requestPeers(peerID, Port)
	if err != nil {
		return err
	}

	torrent := client.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}
	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	err = t.saveToDisk(buf, path)
	if err != nil {
		return err
	}
	return nil
}

func (t *TorrentFile) saveToDisk(buf []byte, path string) error {
	if len(t.Files) == 0 {
		err := os.MkdirAll(path, os.ModePerm) // Create directories recursively if they don't exist
		if err != nil {
			return err
		}

		fullPath := filepath.Join(path, t.Name)
		file, err := os.Create(fullPath)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = file.Write(buf)
		if err != nil {
			return err
		}
		return nil
	}
	fullPath := filepath.Join(path, t.Name)
	startIndex := 0
	curIndex := 0
	buffer := bytes.NewReader(buf)
	for _, f := range t.Files {
		startIndex = curIndex
		curIndex = startIndex + f.Length
		curPath := filepath.Join(fullPath, filepath.Join(f.Path...))

		err := os.MkdirAll(filepath.Dir(curPath), os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		file, err := os.Create(curPath)
		if err != nil {
			return err
		}
		sectionReader := io.NewSectionReader(buffer, int64(startIndex), int64(f.Length))
		multiWriter := io.MultiWriter(file)
		_, err = io.Copy(multiWriter, sectionReader)
		file.Close()
		if err != nil {
			return err
		}
		log.Printf("Write to %s file \n", curPath)
	}
	return nil
}

func (bto *bencodeTorrent) toTorrentFile(info_bytes []byte) (TorrentFile, error) {
	infoHash := sha1.Sum(info_bytes)
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	length := 0
	if len(bto.Info.Files) > 0 {
		for _, f := range bto.Info.Files {
			length += f.Length
		}
	} else {
		length += bto.Info.Length
	}
	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      length,
		Name:        bto.Info.Name,
		Files:       bto.Info.Files,
	}
	return t, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}
