package keystore

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestKeystoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	password := "test-password-123"

	store, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	addr, err := store.Generate(password)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if addr == (common.Address{}) {
		t.Fatal("empty address returned")
	}
	t.Logf("generated address: %s", addr.Hex())

	// Unlock
	signer, err := store.Unlock(password)
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if signer.Address() != addr {
		t.Fatalf("address mismatch: got %s want %s", signer.Address().Hex(), addr.Hex())
	}

	// Sign a dummy hash and verify
	hash := crypto.Keccak256Hash([]byte("test message")).Bytes()
	sig, err := signer.SignHash(hash)
	if err != nil {
		t.Fatalf("SignHash: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("signature length %d, want 65", len(sig))
	}

	// Recover and verify
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		t.Fatalf("SigToPub: %v", err)
	}
	recovered := crypto.PubkeyToAddress(*pubKey)
	if recovered != addr {
		t.Fatalf("recovered address %s != expected %s", recovered.Hex(), addr.Hex())
	}
}

func TestGenerateRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	password := "test-pw"

	store, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := store.Generate(password); err != nil {
		t.Fatalf("first Generate: %v", err)
	}

	// Second generate should fail
	if _, err := store.Generate(password); err == nil {
		t.Fatal("expected error on second Generate, got nil")
	}
}

func TestPasswordEnvVar(t *testing.T) {
	t.Setenv("JIRIKI_KEYSTORE_PASSWORD", "env-password")
	pw, err := ReadPassword("Enter password: ")
	if err != nil {
		t.Fatalf("ReadPassword: %v", err)
	}
	if pw != "env-password" {
		t.Fatalf("got %q, want %q", pw, "env-password")
	}
}
