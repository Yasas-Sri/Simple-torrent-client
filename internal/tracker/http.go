package tracker

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"torrent-client/internal/bencode"
	"torrent-client/internal/metainfo"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func GeneratePeerID() ([20]byte, error) {
	var id [20]byte
	copy(id[:8], "-GT0001-")
	_, err := rand.Read(id[8:])
	return id, err
}

func escapeInfoHash(hash [20]byte) string {
	var out string
	for _, b := range hash {
		out += fmt.Sprintf("%%%02X", b)
	}
	return out
}

func escapeBytes(b []byte) string {
	out := ""
	for _, v := range b {
		out += fmt.Sprintf("%%%02X", v)
	}
	return out
}

func buildAnnounceURL(meta *metainfo.TorrentMeta) (string, error) {
	peerID, err := GeneratePeerID()
	if err != nil {
		return "", err
	}

	u, err := url.Parse(meta.Announce)
	if err != nil {
		return "", err
	}

	q := u.Query()

	q.Set("port", "6881")
	q.Set("uploaded", "0")
	q.Set("downloaded", "0")
	q.Set("left", fmt.Sprintf("%d", meta.Length))
	q.Set("compact", "1")

	rawQuery := q.Encode()
	if rawQuery != "" {
		rawQuery += "&"
	}

	rawQuery += fmt.Sprintf("info_hash=%s&peer_id=%s",
		escapeInfoHash(meta.InfoHash),
		escapeBytes(peerID[:]),
	)

	u.RawQuery = rawQuery
	return u.String(), nil
}

func GetPeers(meta *metainfo.TorrentMeta) ([]Peer, error) {
	url, err := buildAnnounceURL(meta)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	dec := bencode.NewDecoder(body)
	val, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	respDict := val.(bencode.BDict)

	if fail, ok := respDict["failure reason"]; ok {
		return nil, fmt.Errorf("tracker error: %s", string(fail.(bencode.BString)))
	}

	peersVal, ok := respDict["peers"]
	if !ok {
		return nil, fmt.Errorf("tracker response missing peers")
	}

	peersRaw, ok := peersVal.(bencode.BString)
	if !ok {
		return nil, fmt.Errorf("invalid peers format")
	}

	var peers []Peer

	for i := 0; i+6 <= len(peersRaw); i += 6 {
		ip := net.IP(peersRaw[i : i+4])
		port := binary.BigEndian.Uint16(peersRaw[i+4 : i+6])

		peers = append(peers, Peer{
			IP:   ip,
			Port: port,
		})
	}

	return peers, nil
}
