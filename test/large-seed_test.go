package test

import (
	"hash/adler32"
	"io"
	"math/rand"
	"testing"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/internal/testutil"
	"github.com/anacrolix/torrent/metainfo"
	qt "github.com/frankban/quicktest"
)

func TestLargeSeed(t *testing.T) {
	c := qt.New(t)

	data := make([]byte, 146<<20)
	rand.Read(data)
	seederDataChecksum := adler32.Checksum(data)
	t.Logf("seeder data checksum: %x", seederDataChecksum)
	torrentDataSpec := testutil.Torrent{Files: []testutil.File{{Name: "yuge.bin", Data: string(data)}}}

	info := torrentDataSpec.Info(512 << 10)
	//spew.Dump(info)
	metaInfo := metainfo.MetaInfo{}
	metaInfo.InfoBytes = bencode.MustMarshal(info)

	seederConfig := torrent.TestingConfig(c)
	seederConfig.Seed = true
	seederConfig.Debug = true
	seederConfig.DefaultStorage = testutil.StorageClient{torrentDataSpec}
	seeder, err := torrent.NewClient(seederConfig)
	c.Assert(err, qt.IsNil)
	defer seeder.Close()
	defer testutil.ExportStatusWriter(seeder, "s", c)()

	_, err = seeder.AddTorrent(&metaInfo)
	c.Assert(err, qt.IsNil)

	leecherConfig := torrent.TestingConfig(c)
	//leecherConfig.Debug = true
	leecher, err := torrent.NewClient(leecherConfig)
	c.Assert(err, qt.IsNil)
	defer leecher.Close()
	defer testutil.ExportStatusWriter(leecher, "l", c)()
	leecherTorrent, err := leecher.AddTorrent(&metaInfo)
	c.Assert(err, qt.IsNil)
	added := leecherTorrent.AddClientPeer(seeder)
	t.Logf("added %v peer addrs for seeder to leecher", added)

	//<-leecherTorrent.GotInfo()
	leecherTorrent.DownloadAll()
	ltr := leecherTorrent.NewReader()
	defer ltr.Close()
	leechedData, err := io.ReadAll(ltr)
	c.Assert(err, qt.IsNil)
	leechedDataChecksum := adler32.Checksum(leechedData)
	c.Assert(leechedDataChecksum, qt.Equals, seederDataChecksum)
	t.Log("leeched data passed checksum")
	//select {}
}
