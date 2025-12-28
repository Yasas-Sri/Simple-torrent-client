package metainfo

import (
	"crypto/sha1"
	"errors"

	"torrent-client/internal/bencode"
)

type TorrentMeta struct {
	Announce    string
	Name        string
	PieceLength int64
	Length      int64

	Pieces    [][]byte
	InfoBytes []byte
	InfoHash  [20]byte
}

func ParseTorrent(data []byte) (*TorrentMeta, error) {

	dec := bencode.NewDecoder(data)

	rootVal, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	root, ok := rootVal.(bencode.BDict)
	if !ok {
		return nil, errors.New("torrent file is not a dictionary")
	}

	announceVal, ok := root["announce"]
	if !ok {
		return nil, errors.New("missing announce")
	}
	announce := string(announceVal.(bencode.BString))

	dec = bencode.NewDecoder(data)
	rootRaw, _ := dec.Decode()

	rootDict := rootRaw.(bencode.BDict)

	var infoDict bencode.BDict
	var infoBytes []byte

	for key := range rootDict {
		if key == "info" {

			dec = bencode.NewDecoder(data)
			dict, err := dec.Decode()
			if err != nil {
				return nil, err
			}

			top := dict.(bencode.BDict)
			_ = top

			dec = bencode.NewDecoder(data)
			dec.Decode()
			break
		}
	}

	dec = bencode.NewDecoder(data)
	dec.Decode()

	for dec.Pos() < len(data) && data[dec.Pos()] != 'e' {
		keyVal, _ := dec.Decode()
		key := string(keyVal.(bencode.BString))

		if key == "info" {
			infoDict, infoBytes, err = dec.DecodeDictWithSpan()
			if err != nil {
				return nil, err
			}
			break
		} else {
			dec.Decode()
		}
	}

	if infoDict == nil {
		return nil, errors.New("missing info dictionary")
	}

	name := string(infoDict["name"].(bencode.BString))
	pieceLength := int64(infoDict["piece length"].(bencode.BInt))
	length := int64(infoDict["length"].(bencode.BInt))

	piecesRaw := infoDict["pieces"].(bencode.BString)
	if len(piecesRaw)%20 != 0 {
		return nil, errors.New("invalid pieces length")
	}

	var pieces [][]byte
	for i := 0; i < len(piecesRaw); i += 20 {
		pieces = append(pieces, piecesRaw[i:i+20])
	}

	hash := sha1.Sum(infoBytes)

	return &TorrentMeta{
		Announce:    announce,
		Name:        name,
		PieceLength: pieceLength,
		Length:      length,
		Pieces:      pieces,
		InfoBytes:   infoBytes,
		InfoHash:    hash,
	}, nil

}
