package p2p

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
	"torrent-client/internal/metainfo"
	"torrent-client/internal/peer"
	"torrent-client/internal/storage"
	"torrent-client/internal/tracker"
)

const MaxBlockSize = 16384

type Manager struct {
	Torrents map[string]*Torrent
	mu       sync.RWMutex
	PeerID   [20]byte
}

type peerState struct {
	conn     net.Conn
	choked   bool
	bitfield peer.Bitfield
}

type Torrent struct {
	Peers           []string
	PeerID          [20]byte
	InfoHash        [20]byte
	PieceHashes     [][20]byte
	PieceLength     int
	Length          int
	Name            string
	BytesDownloaded int
}

type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	data  []byte
}

type TorrentStats struct {
	Name        string  `json:"name"`
	Percent     float64 `json:"percent"`
	Downloaded  int     `json:"downloaded"`
	TotalLength int     `json:"totalLength"`
	Peers       int     `json:"peers"`
	InfoHash    string  `json:"infoHash"`
}

func NewManager(myID [20]byte) *Manager {
	return &Manager{
		Torrents: make(map[string]*Torrent),
		PeerID:   myID,
	}
}

func (t *Torrent) Download() error {
	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)

	doneCount := 0
	fmt.Println("Verifying existing files...")
	for index, hash := range t.PieceHashes {
		length := t.calculatePieceSize(index)

		if t.checkPieceOnDisk(index, hash) {
			doneCount++
			t.BytesDownloaded += length
			continue
		}
		workQueue <- &pieceWork{index, hash, length}
	}

	if doneCount > 0 {
		fmt.Printf("Resuming from %.2f%%...\n", float64(doneCount)/float64(len(t.PieceHashes))*100)
	}

	for _, address := range t.Peers {
		go t.startDownloadWorker(address, workQueue, results)
	}

	for doneCount < len(t.PieceHashes) {
		res := <-results
		err := storage.SavePiece(t.Name, res.index, t.PieceLength, res.data)
		if err != nil {
			fmt.Printf("Error saving piece %d: %v\n", res.index, err)
			continue
		}
		t.BytesDownloaded += len(res.data)
		doneCount++
		percent := float64(doneCount) / float64(len(t.PieceHashes)) * 100
		fmt.Printf("\rDownloaded: %d/%d (%.2f%%)", doneCount, len(t.PieceHashes), percent)
	}
	return nil
}

func (t *Torrent) calculatePieceSize(index int) int {
	begin := index * t.PieceLength
	end := begin + t.PieceLength
	if end > t.Length {
		return t.Length - begin
	}
	return t.PieceLength
}

func (t *Torrent) startDownloadWorker(addr string, workQueue chan *pieceWork, results chan *pieceResult) {

	conn, err := t.establishPeer(addr)
	if err != nil {
		return
	}
	defer conn.Close()

	state := &peerState{
		conn:   conn,
		choked: true,
	}

	for pw := range workQueue {

		if state.bitfield != nil && !state.bitfield.HasPiece(pw.index) {
			workQueue <- pw
			continue
		}

		data, err := t.attemptDownload(state, pw)
		if err != nil {
			fmt.Printf("   X Peer %s failed on piece %d: %v\n", addr, pw.index, err)
			workQueue <- pw
			return
		}

		results <- &pieceResult{index: pw.index, data: data}
	}
}

func (t *Torrent) attemptDownload(s *peerState, pw *pieceWork) ([]byte, error) {

	s.conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer s.conn.SetDeadline(time.Time{}) // Reset

	progress := peer.NewPieceProgress(pw.index, pw.length)

	t.sendInterested(s.conn)

	for progress.Downloaded < pw.length {
		if !s.choked {

			for progress.Requested-progress.Downloaded < 5*MaxBlockSize && progress.Requested < pw.length {
				blockSize := MaxBlockSize
				if pw.length-progress.Requested < blockSize {
					blockSize = pw.length - progress.Requested
				}
				t.sendRequest(s.conn, pw.index, progress.Requested, blockSize)
				progress.Requested += blockSize
			}
		}

		msg, err := peer.ReadMessage(s.conn)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			continue
		}

		switch msg.ID {
		case peer.MsgUnchoke:
			s.choked = false
		case peer.MsgChoke:
			s.choked = true
		case peer.MsgHave:
			index := int(binary.BigEndian.Uint32(msg.Payload))
			s.bitfield.SetPiece(index)
		case peer.MsgPiece:

			if err := progress.AddBlock(msg.Payload); err != nil {
				return nil, err
			}
		}
	}

	if err := progress.CheckHash(pw.hash); err != nil {
		return nil, err
	}

	return progress.Buffer, nil
}

func (t *Torrent) establishPeer(addr string) (net.Conn, error) {

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}

	hs := peer.NewHandshake(t.InfoHash, t.PeerID)
	_, err = conn.Write(hs.Serialize())
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = peer.Read(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (t *Torrent) sendInterested(conn net.Conn) error {
	msg := &peer.Message{ID: peer.MsgInterested}
	_, err := conn.Write(msg.Serialize())
	return err
}

func (t *Torrent) sendRequest(conn net.Conn, index, begin, length int) error {
	payload := peer.FormatRequest(index, begin, length)
	msg := &peer.Message{ID: peer.MsgRequest, Payload: payload}
	_, err := conn.Write(msg.Serialize())
	return err
}

func (t *Torrent) checkPieceOnDisk(index int, expectedHash [20]byte) bool {
	data, err := storage.ReadPiece(t.Name, index, t.PieceLength)
	if err != nil {
		return false
	}

	hash := sha1.Sum(data)
	return hash == expectedHash
}

func (m *Manager) AddTorrent(torrentData []byte) error {

	meta, err := metainfo.ParseTorrent(torrentData)
	if err != nil {
		return fmt.Errorf("failed to parse torrent: %w", err)
	}

	peers, err := tracker.GetPeers(meta)
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	myID, err := tracker.GeneratePeerID()
	if err != nil {
		fmt.Printf("Failed to generate Peer ID: %v\n", err)
		return err
	}

	var peerAddresses []string
	for _, p := range peers {
		peerAddresses = append(peerAddresses, fmt.Sprintf("%s:%d", p.IP, p.Port))
	}

	pieceHashes := make([][20]byte, len(meta.Pieces))
	for i, slice := range meta.Pieces {
		if len(slice) != 20 {
			fmt.Printf("Warning: Piece %d hash is not 20 bytes long!\n", i)
			continue
		}
		copy(pieceHashes[i][:], slice)
	}

	infoHashHex := fmt.Sprintf("%x", meta.InfoHash)

	m.mu.RLock()
	if _, exists := m.Torrents[infoHashHex]; exists {
		m.mu.RUnlock()
		return fmt.Errorf("torrent already exists in manager")
	}
	m.mu.RUnlock()

	t := &Torrent{
		Peers:       peerAddresses,
		PeerID:      myID,
		InfoHash:    meta.InfoHash,
		PieceHashes: pieceHashes,
		PieceLength: int(meta.PieceLength),
		Length:      int(meta.Length),
		Name:        meta.Name,
	}

	m.mu.Lock()
	m.Torrents[infoHashHex] = t
	m.mu.Unlock()

	fmt.Printf("Manager: Starting background download for %s\n", t.Name)
	go func() {
		err := t.Download()
		if err != nil {
			fmt.Printf("Manager: Torrent %s failed: %v\n", t.Name, err)
		}
	}()

	return nil
}

func (m *Manager) GetStats() []TorrentStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allStats []TorrentStats
	for _, t := range m.Torrents {
		allStats = append(allStats, t.GetStats())
	}
	return allStats
}

func (t *Torrent) GetStats() TorrentStats {

	var percent float64
	if t.Length > 0 {

		percent = (float64(t.BytesDownloaded) / float64(t.Length)) * 100
	}

	return TorrentStats{
		Name:        t.Name,
		Percent:     percent,
		Downloaded:  t.BytesDownloaded,
		TotalLength: t.Length,
		Peers:       len(t.Peers),
		InfoHash:    fmt.Sprintf("%x", t.InfoHash),
	}
}
