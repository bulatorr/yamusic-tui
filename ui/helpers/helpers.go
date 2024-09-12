package helpers

import (
	"crypto/rand"

	"github.com/dece2183/yamusic-tui/api"
)

func ArtistList(artists []api.Artist) (txt string) {
	for _, a := range artists {
		txt += a.Name + ", "
	}
	if len(txt) > 2 {
		txt = txt[:len(txt)-2]
	}
	return
}

func RandString(n int) string {
	const alphanum = "0123456789abcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}
