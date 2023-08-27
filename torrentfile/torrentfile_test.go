package torrentfile

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden.json files")

func TestOpen(t *testing.T) {
	torrent, err := Open("testdata/debian-12.1.0-amd64-netinst.iso.torrent")
	require.Nil(t, err)
	goldenPath := "testdata/debian-12.1.0-amd64-netinst.iso.torrent.golden.json"
	if *update {
		serialized, err := json.MarshalIndent(torrent, "", "  ")
		require.Nil(t, err)
		os.WriteFile(goldenPath, serialized, 0644)
	}

	expected := TorrentFile{}
	golden, err := os.ReadFile(goldenPath)
	require.Nil(t, err)
	err = json.Unmarshal(golden, &expected)
	require.Nil(t, err)

	assert.Equal(t, expected, torrent)
}

func TestSaveDisk(t *testing.T) {
	torrent, err := Open("testdata/KNOPPIX_V9.1CD-2021-01-25-EN.torrent")
	require.Nil(t, err)
	buf := make([]byte, torrent.Length)
	err = torrent.saveToDisk(buf[:], "path")
	require.Nil(t, err)
}
