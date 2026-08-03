// Package backup implements the manually invoked encrypted backup job.
package backup

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrConfig   = errors.New("invalid backup configuration")
	ErrLockHeld = errors.New("backup lock already held")
	ErrPrereq   = errors.New("backup prerequisite failed")
	ErrPackage  = errors.New("backup packaging failed")
	ErrFinalize = errors.New("backup finalization failed")
	ErrCleanup  = errors.New("backup cleanup failed")
)

var fingerprintPattern = regexp.MustCompile(`^(?i:[0-9a-f]{40}|[0-9a-f]{64})$`)

// Config contains only the inputs required by this single backup job.
type Config struct {
	DatabaseURL       string
	UploadsSource     string
	Destination       string
	PublicKeyPath     string
	Recipient         string
	ApplicationCommit string
}

func ConfigFromEnv(getenv func(string) string) Config {
	return Config{
		DatabaseURL:       getenv("BACKUP_DATABASE_URL"),
		UploadsSource:     getenv("BACKUP_UPLOADS_SOURCE"),
		Destination:       getenv("BACKUP_DESTINATION"),
		PublicKeyPath:     getenv("BACKUP_GPG_PUBLIC_KEY_PATH"),
		Recipient:         getenv("BACKUP_GPG_RECIPIENT"),
		ApplicationCommit: getenv("BACKUP_APP_GIT_COMMIT"),
	}
}

// Validate performs only safe local checks. It intentionally never includes
// caller supplied values in errors because the database URL is confidential.
func (c Config) Validate() error {
	if err := validateDatabaseURL(c.DatabaseURL); err != nil {
		return err
	}
	if !fingerprintPattern.MatchString(c.Recipient) {
		return fmt.Errorf("%w: recipient must be a full fingerprint", ErrConfig)
	}
	for _, path := range []string{c.UploadsSource, c.Destination, c.PublicKeyPath} {
		if path == "" || !filepath.IsAbs(path) {
			return fmt.Errorf("%w: backup paths must be absolute", ErrConfig)
		}
	}
	if pathsOverlap(c.UploadsSource, c.Destination) || pathsOverlap(c.UploadsSource, c.PublicKeyPath) || pathsOverlap(c.Destination, c.PublicKeyPath) {
		return fmt.Errorf("%w: backup paths overlap", ErrConfig)
	}
	if info, err := os.Lstat(c.UploadsSource); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: uploads source is unavailable", ErrConfig)
	}
	if info, err := os.Lstat(c.Destination); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: backup destination is unavailable", ErrConfig)
	}
	if info, err := os.Lstat(c.PublicKeyPath); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: public key is unavailable", ErrConfig)
	}
	return nil
}

func validateDatabaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" || u.User == nil || u.User.Username() == "" {
		return fmt.Errorf("%w: database URL is invalid", ErrConfig)
	}
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" || strings.Contains(database, "/") {
		return fmt.Errorf("%w: database URL is invalid", ErrConfig)
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	left = path.Clean(filepath.ToSlash(left))
	right = path.Clean(filepath.ToSlash(right))
	if left == right {
		return true
	}
	return pathWithin(left, right) || pathWithin(right, left)
}

func pathWithin(parent, candidate string) bool {
	if parent == "/" {
		return strings.HasPrefix(candidate, "/")
	}
	return strings.HasPrefix(candidate, parent+"/")
}

func safeCommit(value string) string {
	if regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`).MatchString(value) {
		return strings.ToLower(value)
	}
	return "unknown"
}
