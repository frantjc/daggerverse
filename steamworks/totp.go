package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	steamGuardAlphabet = "23456789BCDFGHJKMNPQRTVWXY"
	steamGuardPeriod   = 30
	steamGuardLength   = 5
)

// generateSteamGuardCode generates the Steam Guard code for the base64-encoded
// shared secret of a Steam account at the given time.
func generateSteamGuardCode(sharedSecret string, at time.Time) (string, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sharedSecret))
	if err != nil {
		return "", fmt.Errorf("decode shared secret: %w", err)
	}

	mac := hmac.New(sha1.New, key)
	if err := binary.Write(mac, binary.BigEndian, uint64(at.Unix()/steamGuardPeriod)); err != nil {
		return "", err
	}

	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	code := make([]byte, steamGuardLength)
	for i := range code {
		code[i] = steamGuardAlphabet[value%uint32(len(steamGuardAlphabet))]
		value /= uint32(len(steamGuardAlphabet))
	}

	return string(code), nil
}
