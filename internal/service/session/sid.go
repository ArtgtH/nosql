package session

import (
	"crypto/rand"
	"encoding/hex"
)

const (
	sidBytes  = 16
	sidLength = sidBytes * 2
)

func NewSID() (string, error) {
	buf := make([]byte, sidBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func IsValidSID(s string) bool {
	if len(s) != sidLength {
		return false
	}

	for _, ch := range s {
		switch {
		case ch >= '0' && ch <= '9':
		case ch >= 'a' && ch <= 'f':
		default:
			return false
		}
	}

	return true
}
