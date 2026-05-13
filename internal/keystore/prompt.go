package keystore

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ReadPassword reads a password from the terminal without echoing it.
// If JIRIKI_KEYSTORE_PASSWORD env var is set, it is used directly (for CI).
func ReadPassword(prompt string) (string, error) {
	if pw := os.Getenv("JIRIKI_KEYSTORE_PASSWORD"); pw != "" {
		return pw, nil
	}
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(b), nil
}
