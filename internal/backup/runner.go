package backup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const lockName = ".permatatex-backup.lock"

var (
	renameFile       = os.Rename
	removeAllPath    = os.RemoveAll
	syncDirectoryRun = syncDirectory
)

type CommandRunner interface {
	Run(context.Context, string, []string, []string) ([]byte, error)
}
type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, name string, args, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append([]string{"LC_ALL=C", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}, env...)
	return cmd.Output()
}

type Job struct {
	Config   Config
	Logger   *slog.Logger
	Commands CommandRunner
	Now      func() time.Time
	ID       string
	Accepted func(Accepted)
}

// Accepted is the intentionally narrow lifecycle signal emitted once a job
// has validated its configuration and prerequisites and acquired its lock.
type Accepted struct {
	BackupID string
	Started  time.Time
}

func NewJob(config Config, logger *slog.Logger) Job {
	return Job{Config: config, Logger: logger, Commands: commandRunner{}, Now: time.Now}
}
func BackupID(now time.Time) string {
	return "permatatex-backup-" + now.UTC().Format("20060102T150405Z")
}

type lockHandle struct{ held *advisoryLock }

func (j Job) Run(ctx context.Context) (runErr error) {
	if err := j.Config.Validate(); err != nil {
		return err
	}
	if j.Logger == nil {
		j.Logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	if j.Commands == nil {
		j.Commands = commandRunner{}
	}
	if j.Now == nil {
		j.Now = time.Now
	}
	now := j.Now().UTC()
	id := j.ID
	if id == "" {
		id = BackupID(now)
	} else if _, err := ParseBackupID(id); err != nil {
		return fmt.Errorf("%w: backup ID is invalid", ErrConfig)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	lock, err := acquireLock(filepath.Join(j.Config.Destination, lockName), id, now)
	if err != nil {
		return err
	}
	var cleanup []func() error
	cleanup = append(cleanup, func() error { return releaseLock(lock) })
	defer func() {
		for i := len(cleanup) - 1; i >= 0; i-- {
			if err := cleanup[i](); err != nil {
				runErr = errors.Join(runErr, ErrCleanup)
			}
		}
	}()
	temp, err := os.MkdirTemp("", "permatatex-backup-")
	if err != nil {
		return fmt.Errorf("%w: temporary workspace", ErrPackage)
	}
	cleanup = append(cleanup, func() error { return removeAllPath(temp) })
	gpgHome, err := os.MkdirTemp("", "permatatex-gpg-")
	if err != nil {
		return fmt.Errorf("%w: temporary keyring", ErrPrereq)
	}
	cleanup = append(cleanup, func() error { return removeAllPath(gpgHome) })
	if err := os.Chmod(gpgHome, 0o700); err != nil {
		return fmt.Errorf("%w: temporary keyring", ErrPrereq)
	}
	dbEnv, err := postgresEnv(j.Config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("%w: database connection", ErrConfig)
	}
	if err := j.verifyAndImportKey(ctx, gpgHome); err != nil {
		return err
	}
	for _, name := range []string{"pg_dump", "pg_restore", "pg_dumpall", "psql"} {
		if err := j.command(ctx, name, []string{"--version"}, nil); err != nil {
			return prerequisiteError(ctx, name+" availability")
		}
	}
	version, err := j.commandOutput(ctx, "psql", []string{"--no-password", "-At", "-c", "SHOW server_version"}, dbEnv)
	if err != nil {
		return prerequisiteError(ctx, "postgres version")
	}
	size, err := j.commandOutput(ctx, "psql", []string{"--no-password", "-At", "-c", "SELECT pg_database_size(current_database())"}, dbEnv)
	if err != nil {
		return prerequisiteError(ctx, "database size")
	}
	migration, err := j.commandOutput(ctx, "psql", []string{"--no-password", "-At", "-c", "SELECT version || '|' || dirty FROM schema_migrations ORDER BY version DESC LIMIT 1"}, dbEnv)
	if err != nil {
		return prerequisiteError(ctx, "migration state")
	}
	pgVersion, dbBytes, migrationVersion, dirty, err := sanitizeDatabaseFacts(version, size, migration)
	if err != nil {
		return fmt.Errorf("%w: database metadata", ErrPrereq)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if j.Accepted != nil {
		j.Accepted(Accepted{BackupID: id, Started: now})
	}
	j.Logger.Info("backup started", slog.String("backup_id", id), slog.String("timestamp_utc", now.Format(time.RFC3339)))
	dump := filepath.Join(temp, "database.dump")
	if err := j.command(ctx, "pg_dump", []string{"--no-password", "--format=custom", "--file=" + dump}, dbEnv); err != nil {
		return prerequisiteError(ctx, "database dump")
	}
	if err := j.command(ctx, "pg_restore", []string{"--no-password", "--list", dump}, nil); err != nil {
		return prerequisiteError(ctx, "database dump validation")
	}
	globals := filepath.Join(temp, "globals.sql")
	if err := j.command(ctx, "pg_dumpall", []string{"--no-password", "--globals-only", "--file=" + globals}, dbEnv); err != nil {
		return prerequisiteError(ctx, "globals dump")
	}
	if err := syncExistingFile(dump); err != nil {
		return fmt.Errorf("%w: database dump", ErrPackage)
	}
	if err := syncExistingFile(globals); err != nil {
		return fmt.Errorf("%w: globals dump", ErrPackage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	uploadArchive := filepath.Join(temp, "uploads.tar.gz")
	stats, manifest, err := writeUploadsArchive(ctx, j.Config.UploadsSource, uploadArchive, now)
	if err != nil {
		return fmt.Errorf("%w: uploads archive", ErrPackage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	manifestPath := filepath.Join(temp, "uploads-manifest.sha256")
	if err := writeManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("%w: uploads manifest", ErrPackage)
	}
	metadata := filepath.Join(temp, "metadata.txt")
	metadataText := fmt.Sprintf("backup_id=%s\ntimestamp_utc=%s\napplication_git_commit=%s\npostgresql_version=%s\nmigration_version=%s\nmigration_dirty=%t\ndatabase_size_bytes=%d\nuploads_file_count=%d\nuploads_total_bytes=%d\n", id, now.Format(time.RFC3339), safeCommit(j.Config.ApplicationCommit), pgVersion, migrationVersion, dirty, dbBytes, stats.Files, stats.Bytes)
	if err := writeSyncedFile(metadata, []byte(metadataText)); err != nil {
		return fmt.Errorf("%w: metadata", ErrPackage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	components := []string{dump, globals, uploadArchive, manifestPath, metadata}
	checksums := filepath.Join(temp, "SHA256SUMS")
	if err := writeChecksums(ctx, checksums, components); err != nil {
		return fmt.Errorf("%w: checksums", ErrPackage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	archive := filepath.Join(temp, id+".tar.gz")
	if err := writeTarGz(ctx, archive, id, append(components, checksums), now); err != nil {
		return fmt.Errorf("%w: package archive", ErrPackage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	partialPackage := filepath.Join(j.Config.Destination, id+".tar.gz.gpg.partial")
	partialSidecar := filepath.Join(j.Config.Destination, id+".tar.gz.gpg.sha256.partial")
	finalPackage := filepath.Join(j.Config.Destination, id+".tar.gz.gpg")
	finalSidecar := finalPackage + ".sha256"
	cleanup = append(cleanup, func() error { return removeIfExists(partialPackage) }, func() error { return removeIfExists(partialSidecar) })
	if err := j.encrypt(ctx, gpgHome, archive, partialPackage); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := syncExistingFile(partialPackage); err != nil {
		return fmt.Errorf("%w: encrypted package", ErrPackage)
	}
	sum, err := fileSHA256(ctx, partialPackage)
	if err != nil {
		return fmt.Errorf("%w: encrypted checksum", ErrPackage)
	}
	verified, err := fileSHA256(ctx, partialPackage)
	if err != nil || verified != sum {
		return fmt.Errorf("%w: encrypted checksum verification", ErrPackage)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writeSyncedFile(partialSidecar, []byte(sum+"  "+filepath.Base(finalPackage)+"\n")); err != nil {
		return fmt.Errorf("%w: encrypted checksum", ErrPackage)
	}
	if err := finalize(ctx, partialPackage, partialSidecar, finalPackage, finalSidecar, j.Config.Destination); err != nil {
		return err
	}
	retentionToken, err := randomToken()
	if err != nil {
		return fmt.Errorf("%w: retention", ErrFinalize)
	}
	finalizationCtx := context.WithoutCancel(ctx)
	removed, kept, err := applyRetention(finalizationCtx, j.Config.Destination, filepath.Base(finalPackage), retentionToken)
	if err != nil {
		return err
	}
	j.Logger.Info("backup completed", slog.String("backup_id", id), slog.Int64("uploads_files", stats.Files), slog.Int64("uploads_bytes", stats.Bytes), slog.String("postgres_major", strings.Split(pgVersion, ".")[0]), slog.String("migration_version", migrationVersion), slog.Bool("migration_dirty", dirty), slog.Int("retention_removed", removed), slog.Int("retention_kept", kept))
	return nil
}

func (j Job) command(ctx context.Context, name string, args, env []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := j.Commands.Run(ctx, name, args, env)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
func (j Job) commandOutput(ctx context.Context, name string, args, env []string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := j.Commands.Run(ctx, name, args, env)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, err
}
func prerequisiteError(ctx context.Context, stage string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: %s", ErrPrereq, stage)
}

func acquireLock(path, id string, started time.Time) (lockHandle, error) {
	held, err := openAdvisoryLock(path)
	if errors.Is(err, ErrCleanup) {
		return lockHandle{}, errors.Join(fmt.Errorf("%w: lock", ErrFinalize), ErrCleanup)
	}
	if errors.Is(err, ErrLockHeld) {
		return lockHandle{}, ErrLockHeld
	}
	if err != nil {
		return lockHandle{}, fmt.Errorf("%w: lock", ErrFinalize)
	}
	lock := lockHandle{held: held}
	if err := writeLock(held.file, id, started); err != nil {
		if releaseErr := releaseLock(lock); releaseErr != nil {
			return lockHandle{}, errors.Join(fmt.Errorf("%w: lock", ErrFinalize), ErrCleanup)
		}
		return lockHandle{}, fmt.Errorf("%w: lock", ErrFinalize)
	}
	return lock, nil
}
func writeLock(f *os.File, id string, started time.Time) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	_, err := fmt.Fprintf(f, "backup_id=%s\nstarted_utc=%s\n", id, started.UTC().Format(time.RFC3339))
	return errors.Join(err, f.Sync())
}
func releaseLock(lock lockHandle) error {
	if lock.held == nil || releaseAdvisoryLock(lock.held) != nil {
		return fmt.Errorf("%w: lock cleanup", ErrFinalize)
	}
	return nil
}

func randomToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
func removeIfExists(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: cleanup", ErrFinalize)
	}
	return nil
}

func postgresEnv(raw string) ([]string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		host, port = u.Host, "5432"
	}
	database := strings.TrimPrefix(u.Path, "/")
	if host == "" || database == "" || strings.Contains(database, "/") {
		return nil, errors.New("invalid database")
	}
	password, _ := u.User.Password()
	env := []string{"PGHOST=" + host, "PGPORT=" + port, "PGUSER=" + u.User.Username(), "PGDATABASE=" + database, "PGPASSWORD=" + password}
	for query, envName := range map[string]string{"sslmode": "PGSSLMODE", "sslrootcert": "PGSSLROOTCERT", "sslcert": "PGSSLCERT", "sslkey": "PGSSLKEY", "connect_timeout": "PGCONNECT_TIMEOUT", "target_session_attrs": "PGTARGETSESSIONATTRS"} {
		if value := u.Query().Get(query); value != "" {
			env = append(env, envName+"="+value)
		}
	}
	return env, nil
}

func (j Job) verifyAndImportKey(ctx context.Context, home string) error {
	out, err := j.commandOutput(ctx, "gpg", []string{"--batch", "--no-tty", "--no-auto-key-retrieve", "--homedir", home, "--with-colons", "--import-options", "show-only", "--dry-run", "--import", j.Config.PublicKeyPath}, nil)
	if err != nil {
		return prerequisiteError(ctx, "public key inspection")
	}
	if !validPublicKeyListing(string(out), j.Config.Recipient) {
		return fmt.Errorf("%w: mounted key is not an unambiguous public recipient", ErrPrereq)
	}
	if err := j.command(ctx, "gpg", []string{"--batch", "--no-tty", "--no-auto-key-retrieve", "--homedir", home, "--import", j.Config.PublicKeyPath}, nil); err != nil {
		return prerequisiteError(ctx, "public key import")
	}
	return nil
}
func validPublicKeyListing(output, recipient string) bool {
	public, primary := 0, ""
	expectFingerprint := ""
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 10 || fields[0] == "" {
			return false
		}
		switch fields[0] {
		case "sec", "ssb":
			return false
		case "pub":
			if public != 0 || expectFingerprint != "" || primary != "" {
				return false
			}
			public++
			expectFingerprint = "primary"
		case "sub":
			if public != 1 || primary == "" || expectFingerprint != "" {
				return false
			}
			expectFingerprint = "subkey"
		case "fpr":
			if expectFingerprint == "" || !fingerprintPattern.MatchString(fields[9]) {
				return false
			}
			if expectFingerprint == "primary" {
				primary = fields[9]
			}
			expectFingerprint = ""
		case "uid", "uat", "sig", "rev", "rvk":
			if public != 1 || primary == "" || expectFingerprint != "" {
				return false
			}
		case "tru":
			if public != 0 || expectFingerprint != "" {
				return false
			}
		default:
			return false
		}
	}
	return public == 1 && expectFingerprint == "" && fingerprintPattern.MatchString(primary) && strings.EqualFold(primary, recipient)
}
func (j Job) encrypt(ctx context.Context, home, input, output string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := j.command(ctx, "gpg", []string{"--batch", "--yes", "--no-tty", "--no-auto-key-retrieve", "--homedir", home, "--trust-model", "always", "--recipient", j.Config.Recipient, "--output", output, "--encrypt", input}, nil); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%w: encryption", ErrPackage)
	}
	return nil
}

var backupName = regexp.MustCompile(`^permatatex-backup-(\d{8}T\d{6}Z)\.tar\.gz\.gpg$`)

func finalize(ctx context.Context, partialPackage, partialSidecar, finalPackage, finalSidecar, destination string) error {
	for _, path := range []string{finalPackage, finalSidecar} {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("%w: output already exists", ErrFinalize)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: output check", ErrFinalize)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := renameFile(partialPackage, finalPackage); err != nil {
		return fmt.Errorf("%w: package rename", ErrFinalize)
	}
	if err := ctx.Err(); err != nil {
		return cancellationRollback(err, destination, finalPackage, finalSidecar)
	}
	if err := renameFile(partialSidecar, finalSidecar); err != nil {
		return errors.Join(fmt.Errorf("%w: checksum rename", ErrFinalize), rollbackCurrentPair(destination, finalPackage, finalSidecar))
	}
	if err := ctx.Err(); err != nil {
		return cancellationRollback(err, destination, finalPackage, finalSidecar)
	}
	if err := syncDirectoryRun(destination); err != nil {
		return errors.Join(fmt.Errorf("%w: destination sync", ErrFinalize), rollbackCurrentPair(destination, finalPackage, finalSidecar))
	}
	return nil
}

func cancellationRollback(cancelErr error, destination, finalPackage, finalSidecar string) error {
	if err := rollbackCurrentPair(destination, finalPackage, finalSidecar); err != nil {
		return errors.Join(cancelErr, ErrFinalize)
	}
	return cancelErr
}

func rollbackCurrentPair(destination, finalPackage, finalSidecar string) error {
	return errors.Join(removeIfExists(finalSidecar), removeIfExists(finalPackage), syncDirectoryRun(destination))
}

func syncDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

type retainedPair struct {
	packagePath, sidecarPath string
	at                       time.Time
}

func applyRetention(ctx context.Context, destination, current string, token string) (int, int, error) {
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: retention scan", ErrFinalize)
	}
	var pairs []retainedPair
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}
		if backupName.FindStringSubmatch(entry.Name()) == nil {
			continue
		}
		packagePath := filepath.Join(destination, entry.Name())
		sidecarPath := packagePath + ".sha256"
		id := strings.TrimSuffix(entry.Name(), packageSuffix)
		item, err := inspectCompleted(ctx, destination, id)
		if err != nil {
			continue
		}
		at := item.CompletedAt
		pairs = append(pairs, retainedPair{packagePath, sidecarPath, at})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].at.After(pairs[j].at) })
	dates, kept, removed := map[string]bool{}, 0, 0
	for _, pair := range pairs {
		name := filepath.Base(pair.packagePath)
		if name == current {
			kept++
			continue
		}
		date := pair.at.Format("2006-01-02")
		if !dates[date] && len(dates) < 7 {
			dates[date] = true
			kept++
			continue
		}
		if err := retirePair(ctx, pair, token); err != nil {
			return removed, kept, err
		}
		removed++
	}
	return removed, kept, nil
}
func retirePair(ctx context.Context, pair retainedPair, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	packageTomb := pair.packagePath + ".retention-" + token + ".partial"
	sidecarTomb := pair.sidecarPath + ".retention-" + token + ".partial"
	if err := renameFile(pair.packagePath, packageTomb); err != nil {
		return fmt.Errorf("%w: retention rename", ErrFinalize)
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, renameFile(packageTomb, pair.packagePath))
	}
	if err := renameFile(pair.sidecarPath, sidecarTomb); err != nil {
		return errors.Join(fmt.Errorf("%w: retention rename", ErrFinalize), renameFile(packageTomb, pair.packagePath))
	}
	if err := os.Remove(packageTomb); err != nil {
		return fmt.Errorf("%w: retention delete", ErrFinalize)
	}
	if err := os.Remove(sidecarTomb); err != nil {
		return fmt.Errorf("%w: retention delete", ErrFinalize)
	}
	return nil
}

func sanitizeDatabaseFacts(version, size, migration []byte) (string, int64, string, bool, error) {
	pgVersion := strings.TrimSpace(string(version))
	if !regexp.MustCompile(`^[0-9]+(?:\.[0-9]+){0,2}$`).MatchString(pgVersion) {
		return "", 0, "", false, errors.New("version")
	}
	dbBytes, err := strconv.ParseInt(strings.TrimSpace(string(size)), 10, 64)
	if err != nil || dbBytes < 0 {
		return "", 0, "", false, errors.New("size")
	}
	migrationVersion, dirty, err := parseMigrationState(migration)
	if err != nil {
		return "", 0, "", false, errors.New("migration")
	}
	return pgVersion, dbBytes, migrationVersion, dirty, nil
}

func parseMigrationState(output []byte) (string, bool, error) {
	row := strings.TrimSpace(string(output))
	if row == "" || strings.ContainsAny(row, "\r\n") {
		return "", false, errors.New("migration row")
	}
	fields := strings.Split(row, "|")
	if len(fields) != 2 || fields[0] == "" || fields[1] == "" {
		return "", false, errors.New("migration fields")
	}
	version, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || version < 0 {
		return "", false, errors.New("migration version")
	}
	dirty, err := strconv.ParseBool(fields[1])
	if err != nil {
		return "", false, errors.New("migration dirty")
	}
	return strconv.FormatInt(version, 10), dirty, nil
}
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrCleanup), errors.Is(err, ErrFinalize):
		return 6
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return 130
	case errors.Is(err, ErrConfig):
		return 2
	case errors.Is(err, ErrLockHeld):
		return 3
	case errors.Is(err, ErrPrereq):
		return 4
	default:
		return 5
	}
}
