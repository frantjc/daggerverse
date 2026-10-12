package main

import (
	"strings"
	"testing"
	"time"
)

func TestSteamGuardCode(t *testing.T) {
	const secret = "AAAAAAAAAAAAAAAAAAAAAAAAAAA="

	at := time.Unix(1_700_000_000, 0)

	code, err := generateSteamGuardCode(secret, at)
	if err != nil {
		t.Fatal(err)
	}

	if len(code) != steamGuardLength {
		t.Fatalf("expected %d characters, got %q", steamGuardLength, code)
	}

	for _, c := range code {
		if !strings.ContainsRune(steamGuardAlphabet, c) {
			t.Fatalf("unexpected character %q in %q", c, code)
		}
	}

	same, err := generateSteamGuardCode(secret, at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	if same != code {
		t.Fatalf("expected the same code within a period, got %q and %q", code, same)
	}

	if _, err := generateSteamGuardCode("not base64!", at); err == nil {
		t.Fatal("expected an error for an invalid shared secret")
	}
}
