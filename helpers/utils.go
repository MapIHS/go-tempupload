package helpers

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
