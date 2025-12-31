package metainfo

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"torrent-client/internal/bencode"
)

type TorrentMeta struct {
	Announce    string
	Name        string
	PieceLength int64
	Length      int64
	Pieces      [][]byte
	InfoBytes   []byte
	InfoHash    [20]byte
}

func ParseTorrent(data []byte) (*TorrentMeta, error) {
	dec := bencode.NewDecoder(data)

	if dec.Peek() != 'd' {
		return nil, fmt.Errorf("invalid torrent: root must be a dictionary")
	}
	dec.PosIncr(1)

	var announce string
	var infoDict bencode.BDict
	var infoBytes []byte

	for dec.Peek() != 'e' && dec.Peek() != 0 {
		keyVal, err := dec.Decode()
		if err != nil {
			return nil, err
		}

		key := string(keyVal.(bencode.BString))

		switch key {
		case "announce":
			val, err := dec.Decode()
			if err != nil {
				return nil, err
			}
			announce = string(val.(bencode.BString))
		case "info":
			dict, raw, err := dec.DecodeDictWithSpan()
			if err != nil {
				return nil, err
			}
			infoDict = dict
			infoBytes = raw
		default:

			if err := dec.SkipValue(); err != nil {
				return nil, err
			}
		}
	}

	if infoDict == nil {
		return nil, errors.New("missing info dictionary")
	}

	var totalLength int64
	if val, ok := infoDict["length"]; ok {
		totalLength = int64(val.(bencode.BInt))
	} else if files, ok := infoDict["files"]; ok {
		for _, f := range files.(bencode.BList) {
			fileDict := f.(bencode.BDict)
			totalLength += int64(fileDict["length"].(bencode.BInt))
		}
	}

	piecesRaw := infoDict["pieces"].(bencode.BString)
	if len(piecesRaw)%20 != 0 {
		return nil, errors.New("invalid pieces length")
	}
	var pieces [][]byte
	for i := 0; i < len(piecesRaw); i += 20 {
		pieces = append(pieces, piecesRaw[i:i+20])
	}

	return &TorrentMeta{
		Announce:    announce,
		Name:        string(infoDict["name"].(bencode.BString)),
		PieceLength: int64(infoDict["piece length"].(bencode.BInt)),
		Length:      totalLength,
		Pieces:      pieces,
		InfoBytes:   infoBytes,
		InfoHash:    sha1.Sum(infoBytes),
	}, nil
}
