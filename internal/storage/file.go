package storage

import (
	"os"
)

func SavePiece(path string, index int, pieceLength int, data []byte) error {

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	offset := int64(index) * int64(pieceLength)
	_, err = f.WriteAt(data, offset)
	return err
}

func ReadPiece(path string, index int, pieceLength int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	offset := int64(index) * int64(pieceLength)
	data := make([]byte, pieceLength)
	_, err = f.ReadAt(data, offset)
	if err != nil {
		return nil, err
	}
	return data, nil
}
