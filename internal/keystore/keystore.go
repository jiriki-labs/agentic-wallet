package keystore

import (
	"fmt"
	"os"
	"path/filepath"

	gokeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const scryptN = 262144 // N parameter for scrypt (2^18)
const scryptP = 1

// Store wraps the go-ethereum keystore.
type Store struct {
	ks  *gokeystore.KeyStore
	dir string
}

// New opens (or creates) a keystore backed by dir.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("mkdir keystore dir: %w", err)
	}
	ks := gokeystore.NewKeyStore(dir, scryptN, scryptP)
	return &Store{ks: ks, dir: dir}, nil
}

// Generate creates a new account and writes it to the keystore directory.
// Returns an error if an account already exists (single-key enforcement).
func (s *Store) Generate(password string) (common.Address, error) {
	if accs := s.ks.Accounts(); len(accs) > 0 {
		return common.Address{}, fmt.Errorf("keystore already contains an account at %s; remove it first or use a different keystore dir", s.dir)
	}
	acc, err := s.ks.NewAccount(password)
	if err != nil {
		return common.Address{}, fmt.Errorf("generate account: %w", err)
	}
	return acc.Address, nil
}

// Address returns the first account address in the keystore.
func (s *Store) Address() (common.Address, error) {
	accs := s.ks.Accounts()
	if len(accs) == 0 {
		return common.Address{}, fmt.Errorf("no accounts in keystore at %s", s.dir)
	}
	return accs[0].Address, nil
}

// Unlock decrypts the first account using the given password and returns a Signer.
func (s *Store) Unlock(password string) (*Signer, error) {
	accs := s.ks.Accounts()
	if len(accs) == 0 {
		return nil, fmt.Errorf("no accounts in keystore at %s", s.dir)
	}
	acc := accs[0]
	if err := s.ks.Unlock(acc, password); err != nil {
		return nil, fmt.Errorf("unlock keystore: %w", err)
	}
	// Export key to get the raw private key for signing
	keyJSON, err := s.ks.Export(acc, password, password)
	if err != nil {
		return nil, fmt.Errorf("export key: %w", err)
	}
	key, err := gokeystore.DecryptKey(keyJSON, password)
	if err != nil {
		return nil, fmt.Errorf("decrypt key: %w", err)
	}
	return &Signer{key: key}, nil
}

// Signer holds an unlocked private key for signing operations.
type Signer struct {
	key *gokeystore.Key
}

// Address returns the signer's Ethereum address.
func (s *Signer) Address() common.Address {
	return s.key.Address
}

// SignHash signs a 32-byte hash with the private key.
func (s *Signer) SignHash(hash []byte) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash must be 32 bytes, got %d", len(hash))
	}
	sig, err := crypto.Sign(hash, s.key.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	return sig, nil
}

// KeystoreFile returns the path to the keystore JSON file for this account.
func (s *Store) KeystoreFile() (string, error) {
	accs := s.ks.Accounts()
	if len(accs) == 0 {
		return "", fmt.Errorf("no accounts")
	}
	// go-ethereum stores the URL as file:// path
	url := accs[0].URL.Path
	if url == "" {
		// Fall back to listing files
		entries, err := os.ReadDir(s.dir)
		if err != nil {
			return "", err
		}
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == "" {
				return filepath.Join(s.dir, e.Name()), nil
			}
		}
		return "", fmt.Errorf("keystore file not found in %s", s.dir)
	}
	return url, nil
}
