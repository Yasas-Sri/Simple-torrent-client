package main

import (
	"fmt"
	"os"
	"torrent-client/internal/bencode"
)

func main() {
	data, _ := os.ReadFile(os.Args[1])
	dec := bencode.NewDecoder(data)
	val, err := dec.Decode()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", val)
}
