package tracker

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

const clientPort = 6881

type Tracker struct {
	Client         *http.Client
	Announce       *url.URL
	BackupAnnounce []*url.URL
}

type TrackerResponse struct {
	PeersString string `bencode:"peers"`
	Interval    int    `bencode:"interval"`
	Complete    int    `bencode:"complete"`
	Incomplete  int    `bencode:"incomplete"`
}

// NewTracker creates a new tracker instance.
func NewTracker(announce string, backupAnnounce []string) (*Tracker, error) {
	announceURL, err := url.Parse(announce)
	if err != nil {
		return nil, fmt.Errorf("invalid announce URL: %s", err)
	}
	backupAnnounceURLs := []*url.URL{}
	for _, backupAnnounce := range backupAnnounce {
		backupAnnounceURL, err := url.Parse(backupAnnounce)
		if err != nil {
			continue
		}
		backupAnnounceURLs = append(backupAnnounceURLs, backupAnnounceURL)
	}
	tracker := &Tracker{
		Client: &http.Client{
			Timeout: time.Second * 10,
		},
		Announce:       announceURL,
		BackupAnnounce: backupAnnounceURLs,
	}
	return tracker, nil
}

// Add parameters to the announce URL.
func (t *Tracker) InitParams(infoHash [20]byte, peerId [20]byte, size int) {
	queryParams := url.Values{}
	// 20 byte sha1 has of bencoded info from metainfo.
	queryParams.Set("info_hash", string(infoHash[:]))
	// String of length 20 which downloader uses as ID.
	queryParams.Set("peer_id", string(peerId[:]))
	// Port number peer is listening on.
	queryParams.Set("port", strconv.Itoa(clientPort))
	// Total amount uploaded so far, encoded in base 10 ascii.
	queryParams.Set("uploaded", "0")
	// Total amount downloaded so far, encoded in base 10 ascii.
	queryParams.Set("downloaded", "0")
	// Number of bytes peer still has to download, encoded in base 10 ascii.
	queryParams.Set("left", strconv.Itoa(size))
	// We want the compact string response.
	queryParams.Set("compact", "0")
	t.Announce.RawQuery = queryParams.Encode()
}

// Sends request to tracker, parses response returns string
// version of a peer list.
func (t *Tracker) RequestPeers() (string, error) {
	resp, err := t.Client.Get(t.Announce.String())
	if err != nil {
		return "", fmt.Errorf("error making request to tracker: %s", err)
	}
	defer resp.Body.Close()
	trackerResponse := TrackerResponse{}
	if err = bencode.Unmarshal(resp.Body, &trackerResponse); err != nil {
		return "", fmt.Errorf("error decoding tracker response: %s", err)
	}
	if len(trackerResponse.PeersString)%6 != 0 {
		return "", fmt.Errorf("invalid peers string: %s", trackerResponse.PeersString)
	}
	return trackerResponse.PeersString, nil
}
