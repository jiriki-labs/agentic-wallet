package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Dir returns the Jiriki config directory.
// Override with $JIRIKI_HOME.
func Dir() string {
	if h := os.Getenv("JIRIKI_HOME"); h != "" {
		return h
	}
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "jiriki")
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "jiriki")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "jiriki")
}

// KeystoreDir returns the keystore subdirectory.
func KeystoreDir() string {
	return filepath.Join(Dir(), "keystore")
}

// AuthFile returns the path to the bearer token file.
func AuthFile() string {
	return filepath.Join(Dir(), "auth")
}

// SocketPath returns the default Unix socket path.
func SocketPath() string {
	return filepath.Join(Dir(), "jiriki.sock")
}

// AuditDB returns the SQLite audit log path.
func AuditDB() string {
	return filepath.Join(Dir(), "audit.db")
}

// PolicyFile returns the default policy YAML path.
func PolicyFile() string {
	return filepath.Join(Dir(), "policy.yaml")
}

// EnsureDir creates the config directory with mode 0700.
func EnsureDir() error {
	d := Dir()
	if err := os.MkdirAll(d, 0700); err != nil {
		return fmt.Errorf("create config dir %s: %w", d, err)
	}
	// Enforce 0700 on existing dirs too
	if err := os.Chmod(d, 0700); err != nil {
		return fmt.Errorf("chmod config dir: %w", err)
	}
	return nil
}

// WriteAuthToken writes the bearer token file with mode 0600.
func WriteAuthToken(token string) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	path := AuthFile()
	return os.WriteFile(path, []byte(token), 0600)
}

// CheckAuthPermissions returns an error if the auth file or its parent
// directory have permissions broader than 0600/0700 respectively.
func CheckAuthPermissions() error {
	authPath := AuthFile()
	dirPath := Dir()

	// Check directory
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		// Dir doesn't exist yet; that's fine for a first-run check
		return nil
	}
	if dirInfo.Mode().Perm()&^os.FileMode(0700) != 0 {
		return fmt.Errorf("config directory %s has unsafe permissions %v (expected 0700 or tighter)", dirPath, dirInfo.Mode().Perm())
	}

	// Check auth file (only if it exists)
	authInfo, err := os.Stat(authPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat auth file: %w", err)
	}
	if authInfo.Mode().Perm()&^os.FileMode(0600) != 0 {
		return fmt.Errorf("auth file %s has unsafe permissions %v (expected 0600 or tighter); run: chmod 0600 %s", authPath, authInfo.Mode().Perm(), authPath)
	}
	return nil
}
