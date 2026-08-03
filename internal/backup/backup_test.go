package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testFingerprint = "0123456789ABCDEF0123456789ABCDEF01234567"

func testConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	uploads, destination := filepath.Join(root, "uploads"), filepath.Join(root, "backups")
	if err := os.MkdirAll(uploads, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(root, "public.asc")
	if err := os.WriteFile(key, []byte("not a real key"), 0o600); err != nil {
		t.Fatal(err)
	}
	return Config{DatabaseURL: "postgres://backup:secret@example.test:5432/inventory?sslmode=require", UploadsSource: uploads, Destination: destination, PublicKeyPath: key, Recipient: testFingerprint, ApplicationCommit: "ffe4e543eb3e47d55f5ba64e46e7f46e39a6d2c4"}
}

func TestConfigValidationAndPathSafety(t *testing.T) {
	cfg := testConfig(t)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	badURL := cfg
	badURL.DatabaseURL = "postgres://user:super-secret@example.test"
	if err := badURL.Validate(); !errors.Is(err, ErrConfig) || strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("unsafe URL error: %v", err)
	}
	notAbsolute := cfg
	notAbsolute.UploadsSource = "uploads"
	if !errors.Is(notAbsolute.Validate(), ErrConfig) {
		t.Fatal("relative source accepted")
	}
	overlapped := cfg
	overlapped.Destination = cfg.UploadsSource
	if !errors.Is(overlapped.Validate(), ErrConfig) {
		t.Fatal("overlapping paths accepted")
	}
	for _, paths := range [][2]string{
		{"/srv/uploads", "/srv/uploads"},
		{"/srv/uploads", "/srv/uploads/backups"},
		{"/srv/uploads/nested", "/srv/uploads"},
	} {
		if !pathsOverlap(paths[0], paths[1]) {
			t.Fatalf("overlap not detected: %#v", paths)
		}
	}
	badFingerprint := cfg
	badFingerprint.Recipient = "short"
	if !errors.Is(badFingerprint.Validate(), ErrConfig) {
		t.Fatal("short recipient accepted")
	}
}

func TestLockAndBackupID(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, lockName)
	at := time.Date(2026, 8, 3, 5, 4, 3, 0, time.UTC)
	if got, want := BackupID(at), "permatatex-backup-20260803T050403Z"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	lock, err := acquireLock(path, "id", at)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireLock(path, "other", at); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("second lock: %v", err)
	}
	if err := releaseLock(lock); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("lock pathname was removed during release")
	}
	later, err := acquireLock(path, "later", at.Add(time.Minute))
	if err != nil {
		t.Fatalf("later lock acquisition failed: %v", err)
	}
	if err := releaseLock(later); err != nil {
		t.Fatal(err)
	}
}

func TestJobRejectsSecondRunBeforeCommands(t *testing.T) {
	cfg := testConfig(t)
	at := time.Date(2026, 8, 3, 5, 4, 3, 0, time.UTC)
	lock, err := acquireLock(filepath.Join(cfg.Destination, lockName), "existing", at)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseLock(lock)
	runner := &countingCommands{}
	job := NewJob(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	job.Commands, job.Now = runner, func() time.Time { return at }
	if err := job.Run(context.Background()); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("expected lock rejection, got %v", err)
	}
	if runner.calls != 0 {
		t.Fatal("second job performed external work")
	}
}

func TestUploadsStatsManifestAndSymlinkRejection(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "b.txt"), []byte("de"), 0o600); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "uploads.tar.gz")
	stats, entries, err := writeUploadsArchive(context.Background(), root, archive, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Files != 2 || stats.Bytes != 5 || len(entries) != 2 {
		t.Fatalf("unexpected stats: %#v %#v", stats, entries)
	}
	manifest := filepath.Join(t.TempDir(), "manifest")
	if err := writeManifest(manifest, entries); err != nil {
		t.Fatal(err)
	}
	if text, _ := os.ReadFile(manifest); !strings.Contains(string(text), "nested/b.txt") {
		t.Fatal("relative manifest missing")
	}
	if err := os.Symlink(filepath.Join(root, "a.txt"), filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if _, _, err := writeUploadsArchive(context.Background(), root, archive, time.Now()); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestUploadsAndPackageTarStructure(t *testing.T) {
	root := t.TempDir()
	uploads := filepath.Join(root, "uploads")
	if err := os.Mkdir(uploads, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "a.txt"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	uploadArchive := filepath.Join(root, "uploads.tar.gz")
	if _, _, err := writeUploadsArchive(context.Background(), uploads, uploadArchive, now); err != nil {
		t.Fatal(err)
	}
	if names := tarNames(t, uploadArchive); !contains(names, "uploads/a.txt") {
		t.Fatalf("uploads structure: %v", names)
	}
	component := filepath.Join(root, "metadata.txt")
	if err := os.WriteFile(component, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(root, "package.tar.gz")
	if err := writeTarGz(context.Background(), pkg, "permatatex-backup-20260803T000000Z", []string{component}, now); err != nil {
		t.Fatal(err)
	}
	if names := tarNames(t, pkg); !contains(names, "permatatex-backup-20260803T000000Z/metadata.txt") {
		t.Fatalf("package structure: %v", names)
	}
}

func tarNames(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var names []string
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return names
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, h.Name)
	}
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestMetadataSanitizationAndCommandFailureClassification(t *testing.T) {
	if safeCommit("bad\nvalue") != "unknown" {
		t.Fatal("unsafe commit retained")
	}
	if safeCommit("AbC1234") != "abc1234" {
		t.Fatal("safe commit changed")
	}
	if _, _, _, _, err := sanitizeDatabaseFacts([]byte("17.2\n"), []byte("42\n"), []byte("41|f\n")); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := sanitizeDatabaseFacts([]byte("unsafe\n"), []byte("42"), []byte("41|f")); err == nil {
		t.Fatal("unsafe metadata accepted")
	}
	if ExitCode(fmtWrapped(ErrPrereq)) != 4 || ExitCode(fmtWrapped(ErrPackage)) != 5 || ExitCode(context.Canceled) != 130 {
		t.Fatal("exit classification failed")
	}
}
func fmtWrapped(err error) error { return errors.Join(errors.New("stage"), err) }

func TestParseMigrationState(t *testing.T) {
	accepted := []struct {
		input   string
		version string
		dirty   bool
	}{
		{input: "41|f", version: "41", dirty: false},
		{input: "41|t", version: "41", dirty: true},
		{input: "41|false", version: "41", dirty: false},
		{input: "41|true", version: "41", dirty: true},
	}
	for _, test := range accepted {
		t.Run(test.input, func(t *testing.T) {
			version, dirty, err := parseMigrationState([]byte(test.input + "\n"))
			if err != nil || version != test.version || dirty != test.dirty {
				t.Fatalf("version=%q dirty=%t err=%v", version, dirty, err)
			}
		})
	}

	rejected := map[string]string{
		"empty":             "",
		"malformed boolean": "41|not-a-boolean",
		"malformed version": "version|false",
		"negative version":  "-1|false",
		"missing field":     "41",
		"additional field":  "41|false|extra",
		"multiple rows":     "41|false\n40|true",
	}
	for name, input := range rejected {
		t.Run(name, func(t *testing.T) {
			if _, _, err := parseMigrationState([]byte(input)); err == nil {
				t.Fatal("invalid migration state accepted")
			}
		})
	}
}

func TestFinalizeIsAtomicAndDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	partialPackage, partialSidecar := filepath.Join(root, "p.partial"), filepath.Join(root, "s.partial")
	finalPackage, finalSidecar := filepath.Join(root, "p.gpg"), filepath.Join(root, "p.gpg.sha256")
	for _, file := range []string{partialPackage, partialSidecar} {
		if err := os.WriteFile(file, []byte(file), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := finalize(context.Background(), partialPackage, partialSidecar, finalPackage, finalSidecar, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(finalPackage); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(partialPackage, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := finalize(context.Background(), partialPackage, partialSidecar, finalPackage, finalSidecar, root); !errors.Is(err, ErrFinalize) {
		t.Fatalf("overwrote output: %v", err)
	}
}

func writeValidPair(t *testing.T, destination, stamp string) {
	t.Helper()
	name := "permatatex-backup-" + stamp + ".tar.gz.gpg"
	body := []byte(stamp)
	if err := os.WriteFile(filepath.Join(destination, name), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(context.Background(), filepath.Join(destination, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, name+".sha256"), []byte(sum+"  "+name+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRetentionDailyPairsOnly(t *testing.T) {
	destination := t.TempDir()
	for day := 1; day <= 9; day++ {
		writeValidPair(t, destination, "202608"+two(day)+"T010000Z")
	}
	writeValidPair(t, destination, "20260809T020000Z") // newest on an already retained day
	if err := os.WriteFile(filepath.Join(destination, "permatatex-baseline-20260801T000000Z.tar.gz.gpg"), []byte("baseline"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "unfamiliar.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "permatatex-backup-20260101T000000Z.tar.gz.gpg"), []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}
	removed, kept, err := applyRetention(context.Background(), destination, "", "test-token")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 3 || kept != 7 {
		t.Fatalf("removed=%d kept=%d", removed, kept)
	}
	for _, name := range []string{"permatatex-baseline-20260801T000000Z.tar.gz.gpg", "unfamiliar.txt", "permatatex-backup-20260101T000000Z.tar.gz.gpg"} {
		if _, err := os.Stat(filepath.Join(destination, name)); err != nil {
			t.Fatalf("unexpected deletion %s", name)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "permatatex-backup-20260809T020000Z.tar.gz.gpg")); err != nil {
		t.Fatal("newest same-day backup deleted")
	}
}
func two(day int) string {
	if day < 10 {
		return "0" + string(rune('0'+day))
	}
	return ""
}

type fakeCommands struct{ fail string }

func (f fakeCommands) Run(_ context.Context, name string, args, _ []string) ([]byte, error) {
	if name == f.fail {
		return nil, errors.New("external failure")
	}
	if name == "gpg" && contains(args, "--with-colons") {
		return []byte("pub:::::::::\nfpr:::::::::" + testFingerprint + "\n"), nil
	}
	if name == "psql" {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "SHOW server_version") {
			return []byte("17.2\n"), nil
		}
		if strings.Contains(joined, "pg_database_size") {
			return []byte("42\n"), nil
		}
		return []byte("41|f\n"), nil
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "--file=") {
			return nil, os.WriteFile(strings.TrimPrefix(arg, "--file="), []byte("dump"), 0o600)
		}
	}
	if name == "gpg" && contains(args, "--encrypt") {
		for i, arg := range args {
			if arg == "--output" {
				return nil, os.WriteFile(args[i+1], []byte("encrypted"), 0o600)
			}
		}
	}
	return nil, nil
}

type recordedCommand struct {
	name string
	args []string
	env  []string
}

type recordingCommands struct {
	calls    []recordedCommand
	delegate CommandRunner
}

func (r *recordingCommands) Run(ctx context.Context, name string, args, env []string) ([]byte, error) {
	r.calls = append(r.calls, recordedCommand{name: name, args: append([]string(nil), args...), env: append([]string(nil), env...)})
	return r.delegate.Run(ctx, name, args, env)
}

type countingCommands struct{ calls int }

func (f *countingCommands) Run(context.Context, string, []string, []string) ([]byte, error) {
	f.calls++
	return nil, errors.New("should not run")
}

func TestSuccessfulJobFinalizesVerifiedEncryptedPair(t *testing.T) {
	cfg := testConfig(t)
	if err := os.WriteFile(filepath.Join(cfg.UploadsSource, "upload.txt"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	job := NewJob(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	job.Commands = fakeCommands{}
	job.Now = func() time.Time { return time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC) }
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	name := "permatatex-backup-20260803T010203Z.tar.gz.gpg"
	if _, err := os.Stat(filepath.Join(cfg.Destination, name)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Destination, name+".sha256")); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(cfg.Destination); err != nil || len(entries) != 3 || entries[0].Name() != lockName {
		t.Fatalf("unexpected final destination entries: %v", entries)
	}
}

func TestFailureCleanupAndSecretsNotLogged(t *testing.T) {
	cfg := testConfig(t)
	secret := "super-secret"
	cfg.DatabaseURL = "postgres://backup:" + secret + "@example.test/inventory"
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	job := NewJob(cfg, logger)
	job.Commands = fakeCommands{fail: "pg_dump"}
	job.Now = func() time.Time { return time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC) }
	err := job.Run(context.Background())
	if !errors.Is(err, ErrPrereq) || strings.Contains(err.Error(), secret) || strings.Contains(logs.String(), secret) {
		t.Fatalf("secret leaked or wrong failure: %v %s", err, logs.String())
	}
	entries, err := os.ReadDir(cfg.Destination)
	if err != nil || len(entries) != 1 || entries[0].Name() != lockName {
		t.Fatalf("incomplete artifacts remain: %v", entries)
	}
}

func TestPublicKeyInspectionColonRecords(t *testing.T) {
	subFingerprint := "89ABCDEF0123456789ABCDEF0123456789ABCDEF"
	publicWithSubkey := strings.Join([]string{
		"pub:-:4096:1:0000000000000000:0:0::::::scESC::::::23::0:",
		"fpr:::::::::" + testFingerprint + ":",
		"uid:-::::0::HASH::Disposable Backup <backup@example.test>::::::::::0:",
		"sub:-:4096:1:1111111111111111:0:0::::::e::::::23:",
		"fpr:::::::::" + subFingerprint + ":",
	}, "\n")
	if !validPublicKeyListing(publicWithSubkey, strings.ToLower(testFingerprint)) {
		t.Fatal("ordinary public key with encryption subkey rejected")
	}
	public := "pub:::::::::\nfpr:::::::::" + testFingerprint + "\n"
	tests := map[string]string{
		"secret only":        "sec:::::::::\nfpr:::::::::" + testFingerprint,
		"mixed secret":       public + "ssb:::::::::\nfpr:::::::::" + subFingerprint,
		"multiple primary":   public + "pub:::::::::\nfpr:::::::::" + subFingerprint,
		"primary mismatch":   "pub:::::::::\nfpr:::::::::AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"fingerprint first":  "fpr:::::::::" + testFingerprint + "\npub:::::::::",
		"uid before primary": "pub:::::::::\nuid:::::::::name\nfpr:::::::::" + testFingerprint,
	}
	for name, listing := range tests {
		t.Run(name, func(t *testing.T) {
			if validPublicKeyListing(listing, testFingerprint) {
				t.Fatal("unsafe or malformed key listing accepted")
			}
		})
	}
}

func TestCurrentPairSurvivesFutureRetentionAndRollback(t *testing.T) {
	destination := t.TempDir()
	current := "permatatex-backup-20260101T000000Z.tar.gz.gpg"
	writeValidPair(t, destination, "20260101T000000Z")
	for day := 2; day <= 8; day++ {
		writeValidPair(t, destination, "202701"+two(day)+"T000000Z")
	}
	if _, _, err := applyRetention(context.Background(), destination, current, "token"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, current)); err != nil {
		t.Fatal("current pair deleted")
	}
	old := "permatatex-backup-20250101T000000Z"
	writeValidPair(t, destination, "20250101T000000Z")
	original := renameFile
	defer func() { renameFile = original }()
	calls := 0
	renameFile = func(a, b string) error {
		calls++
		if calls == 2 {
			return errors.New("injected")
		}
		return original(a, b)
	}
	pair := retainedPair{filepath.Join(destination, old+".tar.gz.gpg"), filepath.Join(destination, old+".tar.gz.gpg.sha256"), time.Now()}
	if err := retirePair(context.Background(), pair, "token"); !errors.Is(err, ErrFinalize) {
		t.Fatal("retention rename failure not classified")
	}
	if _, err := os.Stat(pair.packagePath); err != nil {
		t.Fatal("first retention rename was not rolled back")
	}
}

func TestDatabaseNameAndNoPasswordArguments(t *testing.T) {
	env, err := postgresEnv("postgres://backup:secret@example.test/%69nventory")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(env, "PGDATABASE=inventory") {
		t.Fatal("database name was not decoded")
	}
	if _, err := postgresEnv("postgres://backup:secret@example.test/"); err == nil {
		t.Fatal("root database path accepted")
	}
}

func TestPostgresCommandsUseSafeArgumentsAndDoNotLeakSecrets(t *testing.T) {
	cfg := testConfig(t)
	password := "db-password-unique-value"
	username := "db-actor-unique-value"
	cfg.DatabaseURL = "postgres://" + username + ":" + password + "@example.test/inventory?sslmode=require"
	if err := os.WriteFile(filepath.Join(cfg.UploadsSource, "upload"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	recorder := &recordingCommands{delegate: fakeCommands{}}
	job := NewJob(cfg, slog.New(slog.NewJSONHandler(&logs, nil)))
	job.Commands = recorder
	job.Now = func() time.Time { return time.Date(2026, 8, 3, 1, 2, 5, 0, time.UTC) }
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, call := range recorder.calls {
		switch call.name {
		case "pg_dump", "pg_dumpall", "pg_restore", "psql":
			seen[call.name]++
			if !contains(call.args, "--no-password") {
				t.Fatalf("%s omitted --no-password", call.name)
			}
			joined := strings.Join(call.args, " ")
			for _, secret := range []string{cfg.DatabaseURL, username, password} {
				if strings.Contains(joined, secret) {
					t.Fatalf("%s leaked a database secret in arguments", call.name)
				}
			}
		}
	}
	for _, name := range []string{"pg_dump", "pg_dumpall", "pg_restore", "psql"} {
		if seen[name] == 0 {
			t.Fatalf("%s was not captured", name)
		}
	}
	if strings.Contains(logs.String(), password) || strings.Contains(logs.String(), cfg.DatabaseURL) {
		t.Fatal("database secret leaked to logs")
	}
}

type inspectionFailureCommands struct{ raw string }

func (f inspectionFailureCommands) Run(_ context.Context, name string, args, _ []string) ([]byte, error) {
	if name == "gpg" && contains(args, "--with-colons") {
		return []byte(f.raw), errors.New(f.raw)
	}
	return nil, errors.New("unexpected command")
}

func TestGPGInspectionFailureDoesNotLeakOutputOrFingerprint(t *testing.T) {
	cfg := testConfig(t)
	raw := "raw-gpg-material-" + testFingerprint
	var logs bytes.Buffer
	job := NewJob(cfg, slog.New(slog.NewJSONHandler(&logs, nil)))
	job.Commands = inspectionFailureCommands{raw: raw}
	err := job.Run(context.Background())
	if !errors.Is(err, ErrPrereq) {
		t.Fatalf("unexpected failure: %v", err)
	}
	for _, forbidden := range []string{raw, testFingerprint} {
		if strings.Contains(err.Error(), forbidden) || strings.Contains(logs.String(), forbidden) {
			t.Fatal("GPG inspection detail leaked")
		}
	}
}

type invalidListingCommands struct{ imports int }

func (f *invalidListingCommands) Run(_ context.Context, name string, args, _ []string) ([]byte, error) {
	if name != "gpg" {
		return nil, errors.New("unexpected command")
	}
	if contains(args, "--with-colons") {
		return []byte("sec:::::::::\nfpr:::::::::" + testFingerprint), nil
	}
	f.imports++
	return nil, nil
}

func TestGPGImportDoesNotRunAfterInvalidInspection(t *testing.T) {
	cfg := testConfig(t)
	commands := &invalidListingCommands{}
	job := NewJob(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	job.Commands = commands
	if err := job.Run(context.Background()); !errors.Is(err, ErrPrereq) {
		t.Fatalf("unexpected result: %v", err)
	}
	if commands.imports != 0 {
		t.Fatal("GPG import ran after invalid inspection")
	}
}

func TestCleanupFailureOverridesEarlierClassification(t *testing.T) {
	originalRemoveAll := removeAllPath
	defer func() { removeAllPath = originalRemoveAll }()
	removeAllPath = func(path string) error {
		_ = originalRemoveAll(path)
		return errors.New("injected cleanup failure")
	}

	t.Run("prerequisite", func(t *testing.T) {
		cfg := testConfig(t)
		job := NewJob(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
		job.Commands = fakeCommands{fail: "pg_dump"}
		err := job.Run(context.Background())
		if !errors.Is(err, ErrCleanup) || ExitCode(err) != 6 {
			t.Fatalf("cleanup did not override prerequisite: %v", err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		cfg := testConfig(t)
		if err := os.WriteFile(filepath.Join(cfg.UploadsSource, "upload"), []byte("payload"), 0o600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		job := NewJob(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
		job.Commands = cancellingCommands{cancel: cancel}
		err := job.Run(ctx)
		if !errors.Is(err, ErrCleanup) || ExitCode(err) != 6 {
			t.Fatalf("cleanup did not override cancellation: %v", err)
		}
	})
}

type cancellingCommands struct{ cancel context.CancelFunc }

func (f cancellingCommands) Run(ctx context.Context, name string, args, env []string) ([]byte, error) {
	if name == "gpg" && contains(args, "--encrypt") {
		for i, arg := range args {
			if arg == "--output" {
				if err := os.WriteFile(args[i+1], []byte("encrypted"), 0o600); err != nil {
					return nil, err
				}
				f.cancel()
				return nil, nil
			}
		}
	}
	return fakeCommands{}.Run(ctx, name, args, env)
}

func TestCancellationBeforeFinalizationCleansCurrentArtifacts(t *testing.T) {
	cfg := testConfig(t)
	if err := os.WriteFile(filepath.Join(cfg.UploadsSource, "upload"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	job := NewJob(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	job.Commands = cancellingCommands{cancel}
	job.Now = func() time.Time { return time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC) }
	err := job.Run(ctx)
	if ExitCode(err) != 130 {
		t.Fatalf("got %v", err)
	}
	entries, readErr := os.ReadDir(cfg.Destination)
	if readErr != nil || len(entries) != 1 || entries[0].Name() != lockName {
		t.Fatalf("cancellation left artifacts: %v", entries)
	}
}

func TestFinalizeCancellationBoundaries(t *testing.T) {
	t.Run("immediately before package rename", func(t *testing.T) {
		root, partialPackage, partialSidecar, finalPackage, finalSidecar := finalizationFiles(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := finalize(ctx, partialPackage, partialSidecar, finalPackage, finalSidecar, root); !errors.Is(err, context.Canceled) || ExitCode(err) != 130 {
			t.Fatalf("unexpected result: %v", err)
		}
		assertNoFinalPair(t, finalPackage, finalSidecar)
	})

	t.Run("between package and sidecar rename", func(t *testing.T) {
		root, partialPackage, partialSidecar, finalPackage, finalSidecar := finalizationFiles(t)
		ctx, cancel := context.WithCancel(context.Background())
		original := renameFile
		defer func() { renameFile = original }()
		calls := 0
		renameFile = func(from, to string) error {
			calls++
			err := original(from, to)
			if calls == 1 && err == nil {
				cancel()
			}
			return err
		}
		if err := finalize(ctx, partialPackage, partialSidecar, finalPackage, finalSidecar, root); !errors.Is(err, context.Canceled) || ExitCode(err) != 130 {
			t.Fatalf("unexpected result: %v", err)
		}
		assertNoFinalPair(t, finalPackage, finalSidecar)
	})

	t.Run("after sidecar rename before sync", func(t *testing.T) {
		root, partialPackage, partialSidecar, finalPackage, finalSidecar := finalizationFiles(t)
		ctx, cancel := context.WithCancel(context.Background())
		original := renameFile
		defer func() { renameFile = original }()
		calls := 0
		renameFile = func(from, to string) error {
			calls++
			err := original(from, to)
			if calls == 2 && err == nil {
				cancel()
			}
			return err
		}
		if err := finalize(ctx, partialPackage, partialSidecar, finalPackage, finalSidecar, root); !errors.Is(err, context.Canceled) || ExitCode(err) != 130 {
			t.Fatalf("unexpected result: %v", err)
		}
		assertNoFinalPair(t, finalPackage, finalSidecar)
	})
}

func TestCancellationAfterCommitKeepsPairAndReturnsSuccess(t *testing.T) {
	cfg := testConfig(t)
	if err := os.WriteFile(filepath.Join(cfg.UploadsSource, "upload"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	originalSync := syncDirectoryRun
	defer func() { syncDirectoryRun = originalSync }()
	syncDirectoryRun = func(path string) error {
		err := originalSync(path)
		if err == nil {
			cancel()
		}
		return err
	}
	job := NewJob(cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	job.Commands = fakeCommands{}
	job.Now = func() time.Time { return time.Date(2026, 8, 3, 1, 2, 4, 0, time.UTC) }
	if err := job.Run(ctx); err != nil || ExitCode(err) != 0 {
		t.Fatalf("committed backup returned failure: %v", err)
	}
	name := "permatatex-backup-20260803T010204Z.tar.gz.gpg"
	for _, path := range []string{filepath.Join(cfg.Destination, name), filepath.Join(cfg.Destination, name+".sha256")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("committed artifact missing: %v", err)
		}
	}
}

func TestFinalizeSyncFailureRollsBackCurrentPair(t *testing.T) {
	root, partialPackage, partialSidecar, finalPackage, finalSidecar := finalizationFiles(t)
	originalSync := syncDirectoryRun
	defer func() { syncDirectoryRun = originalSync }()
	calls := 0
	syncDirectoryRun = func(path string) error {
		calls++
		if calls == 1 {
			return errors.New("injected sync failure")
		}
		return originalSync(path)
	}
	err := finalize(context.Background(), partialPackage, partialSidecar, finalPackage, finalSidecar, root)
	if !errors.Is(err, ErrFinalize) || ExitCode(err) != 6 {
		t.Fatalf("sync failure not classified as finalization: %v", err)
	}
	assertNoFinalPair(t, finalPackage, finalSidecar)
}

func finalizationFiles(t *testing.T) (string, string, string, string, string) {
	t.Helper()
	root := t.TempDir()
	partialPackage := filepath.Join(root, "current.gpg.partial")
	partialSidecar := filepath.Join(root, "current.gpg.sha256.partial")
	finalPackage := filepath.Join(root, "current.gpg")
	finalSidecar := finalPackage + ".sha256"
	for _, path := range []string{partialPackage, partialSidecar} {
		if err := os.WriteFile(path, []byte("complete"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root, partialPackage, partialSidecar, finalPackage, finalSidecar
}

func assertNoFinalPair(t *testing.T, finalPackage, finalSidecar string) {
	t.Helper()
	for _, path := range []string{finalPackage, finalSidecar} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("completed artifact remains")
		}
	}
}
