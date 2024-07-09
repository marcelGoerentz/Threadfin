package imgcache

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"path/filepath"
	"strings"
)

// createIndexFromUrl will calculate the URLs md5 and will pick then only the digits within the string too create a key
func createKeyFromUrl(url string) (string) {
  // Berechne den MD5-Hash der URL
  hasher := md5.New()
  io.WriteString(hasher, url)
  return hex.EncodeToString(hasher.Sum(nil))
}

// Faster creation of file names
func createFileNameFromURL(url string, key string) (string) {
	url_stripped := strings.Split(url, "?")[0]
	ext := filepath.Ext(url_stripped)

	var buf bytes.Buffer
	buf.WriteString(key)
	buf.WriteString(ext)

	return buf.String()
}
