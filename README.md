# **BitTorrent-Go** #
A client for the file-sharing protocol BitTorrent.  Will connect to other peers on a peer2peer network to download file specified in a .torrent file.  Information such as peer, activity, download speed and download progress will be displayed on a terminal dashboard.

![Demo](assets/BitTorrentGoDemo.gif)

IP's randomised and detailed errors omitted in demo GIF.
## Installation ##

Clone the repository:
`git clone https://github.com/0xNathanW/bittorrent-go`

Build executable:
`go build main.go`

## Usage ##

To use run the executable with an argument of the path for the .torrent file:

Eg. `{.exe name} {path to .torrent file}`

