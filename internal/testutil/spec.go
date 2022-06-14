package testutil

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/anacrolix/missinggo/expect"
	"github.com/anacrolix/torrent/common"
	"github.com/anacrolix/torrent/segments"
	"github.com/anacrolix/torrent/storage"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

type File struct {
	Name string
	Data string
}

type Torrent struct {
	Files []File
	Name  string
}

func (t *Torrent) IsDir() bool {
	return len(t.Files) == 1 && t.Files[0].Name == ""
}

func (t *Torrent) GetFile(name string) *File {
	if t.IsDir() && t.Name == name {
		return &t.Files[0]
	}
	for _, f := range t.Files {
		if f.Name == name {
			return &f
		}
	}
	return nil
}

func (t *Torrent) Info(pieceLength int64) metainfo.Info {
	info := metainfo.Info{
		Name:        t.Name,
		PieceLength: pieceLength,
	}
	if t.IsDir() {
		info.Length = int64(len(t.Files[0].Data))
	} else {
		for _, f := range t.Files {
			info.Files = append(info.Files, metainfo.FileInfo{
				Path:   []string{f.Name},
				Length: int64(len(f.Data)),
			})
		}
	}
	err := info.GeneratePieces(func(fi metainfo.FileInfo) (io.ReadCloser, error) {
		return ioutil.NopCloser(strings.NewReader(t.GetFile(strings.Join(fi.Path, "/")).Data)), nil
	})
	expect.Nil(err)
	return info
}

func (t *Torrent) Metainfo(pieceLength int64) *metainfo.MetaInfo {
	mi := metainfo.MetaInfo{}
	var err error
	mi.InfoBytes, err = bencode.Marshal(t.Info(pieceLength))
	expect.Nil(err)
	return &mi
}

type StorageClient struct {
	Torrent Torrent
}

var _ storage.ClientImpl = StorageClient{}

type StorageTorrent struct {
	Torrent         Torrent
	info            *metainfo.Info
	fileExtentIndex segments.Index
}

func (s StorageTorrent) piece(p metainfo.Piece) storage.PieceImpl {
	return &storagePiece{&s, p}
}

func (s StorageClient) OpenTorrent(info *metainfo.Info, infoHash metainfo.Hash) (storage.TorrentImpl, error) {
	st := StorageTorrent{
		Torrent:         s.Torrent,
		info:            info,
		fileExtentIndex: segments.NewIndex(common.LengthIterFromUpvertedFiles(info.UpvertedFiles())),
	}
	return storage.TorrentImpl{
		Piece:    st.piece,
		Close:    nil,
		Capacity: nil,
	}, nil
}

type storagePiece struct {
	t *StorageTorrent
	p metainfo.Piece
}

func (s storagePiece) ReadAt(p []byte, off int64) (n int, err error) {
	s.t.fileExtentIndex.Locate(segments.Extent{
		Start:  s.p.Offset() + off,
		Length: int64(len(p)),
	}, func(i int, extent segments.Extent) bool {
		n += copy(p, s.t.Torrent.Files[i].Data[extent.Start:extent.Start+extent.Length])
		return true
	})
	return
}

func (s storagePiece) WriteAt(p []byte, off int64) (n int, err error) {
	panic("unimplemented")
}

func (s storagePiece) MarkComplete() error {
	panic("unimplemented")
}

func (s storagePiece) MarkNotComplete() error {
	panic("unimplemented")
}

func (s storagePiece) Completion() storage.Completion {
	return storage.Completion{true, true}
}
