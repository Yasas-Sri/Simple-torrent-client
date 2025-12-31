package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"torrent-client/internal/metainfo"
	"torrent-client/internal/peer"
	"torrent-client/internal/tracker"
)

func connectToPeer(p tracker.Peer, infoHash, peerID [20]byte) (*peer.Handshake, net.Conn, error) {
	address := net.JoinHostPort(p.IP.String(), fmt.Sprintf("%d", p.Port))

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, nil, err
	}

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	hs := peer.NewHandshake(infoHash, peerID)

	_, err = conn.Write(hs.Serialize())
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	res, err := peer.Read(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to read handshake: %w", err)
	}

	conn.SetDeadline(time.Time{})

	return res, conn, nil
}

func main() {
	if len(os.Args) < 2 {
		panic("usage: torrent <file.torrent>")
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	meta, err := metainfo.ParseTorrent(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("Name:", meta.Name)
	fmt.Println("Tracker:", meta.Announce)
	fmt.Println("Piece length:", meta.PieceLength)
	fmt.Println("Total size:", meta.Length)
	fmt.Println("Pieces:", len(meta.Pieces))
	fmt.Printf("Info hash: %x\n", meta.InfoHash)

	peers, err := tracker.GetPeers(meta)
	if err != nil {
		panic(err)
	}

	fmt.Println("Peers:")
	for _, p := range peers {
		fmt.Printf("  %s:%d\n", p.IP, p.Port)
	}

	myID, err := tracker.GeneratePeerID()
	if err != nil {
		panic(fmt.Errorf("could not generate peer ID: %w", err))
	}

	fmt.Printf("Found %d peers. Starting connection attempts...\n", len(peers))

	var connection net.Conn
	var handshake *peer.Handshake

	for _, p := range peers {
		fmt.Printf("Trying peer: %s:%d\n", p.IP, p.Port)

		handshake, connection, err = connectToPeer(p, meta.InfoHash, myID)
		if err != nil {
			fmt.Printf("Connection failed: %v\n", err)
			continue
		}

		break
	}

	if connection == nil {
		fmt.Println("Could not connect to any peers.")
		return
	}
	defer connection.Close()

	fmt.Printf("Connected! Peer ID: %x\n", handshake.PeerID)

}
