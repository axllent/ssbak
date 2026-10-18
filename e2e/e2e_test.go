//go:build e2e

// Package e2e runs the compiled ssbak binary against a real MySQL/MariaDB
// server: it saves a site to an .sspak, destroys the database and assets,
// restores the .sspak, and verifies everything matches the original.
//
// Set TEST_DB_HOST (plus TEST_DB_PORT, TEST_DB_USER, TEST_DB_PASS and
// TEST_DB_NAME) to run; the tests are skipped otherwise. Each test drops and
// recreates databases named after TEST_DB_NAME, so never point it at data you
// care about. Set SSBAK_BIN to test an existing binary instead of building one.
package e2e

import (
	"archive/tar"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*.sql
var seeds embed.FS

// ssbakBin is the path to the binary under test, set by TestMain.
var ssbakBin string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if bin := os.Getenv("SSBAK_BIN"); bin != "" {
		abs, err := filepath.Abs(bin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		ssbakBin = abs

		return m.Run()
	}

	dir, err := os.MkdirTemp("", "ssbak-e2e-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	ssbakBin = filepath.Join(dir, "ssbak")

	build := exec.Command("go", "build", "-o", ssbakBin, ".")
	build.Dir = ".." // repository root
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr

	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "building ssbak:", err)
		return 1
	}

	return m.Run()
}

// dbConfig holds the connection settings for the server under test.
type dbConfig struct {
	Host, Port, User, Pass, Name string
}

func loadDBConfig(t *testing.T) dbConfig {
	t.Helper()

	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		t.Skip("TEST_DB_HOST not set; skipping e2e tests")
	}

	get := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}

		return fallback
	}

	return dbConfig{
		Host: host,
		Port: get("TEST_DB_PORT", "3306"),
		User: get("TEST_DB_USER", "root"),
		Pass: os.Getenv("TEST_DB_PASS"),
		Name: get("TEST_DB_NAME", "ssbak_e2e"),
	}
}

// open connects to the server, optionally selecting a database.
func (c dbConfig) open(t *testing.T, dbName string) *sql.DB {
	t.Helper()

	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Pass
	cfg.Net = "tcp"
	cfg.Addr = c.Host + ":" + c.Port
	cfg.DBName = dbName
	cfg.MultiStatements = true
	cfg.Params = map[string]string{"charset": "utf8mb4"}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Ping(), "cannot connect to the database server")

	return db
}

// ssbak runs the binary with a clean environment (no SS_* variables leaking in
// from the caller, so the site's .env file is what gets used).
func ssbak(t *testing.T, args ...string) string {
	t.Helper()

	cmd := exec.Command(ssbakBin, args...)
	cmd.Env = cleanEnv()

	out, err := cmd.CombinedOutput()
	t.Logf("ssbak %s\n%s", strings.Join(args, " "), out)
	require.NoError(t, err, "ssbak %s failed", strings.Join(args, " "))

	return string(out)
}

func cleanEnv() []string {
	var env []string

	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "SS_DATABASE_") || strings.HasPrefix(kv, "SS_IGNORE_DOT_ENV=") {
			continue
		}

		env = append(env, kv)
	}

	return env
}

// scenario describes one save/destroy/load/verify cycle.
type scenario struct {
	// seeds are the testdata/*.sql files used to populate the database.
	seeds []string
	// assets also creates, saves and restores a public/assets folder. When
	// false only the database is saved and restored (--db).
	assets bool
	// saveFlags are extra flags for "ssbak save", e.g. --zstd.
	saveFlags []string
	// restrictedRestore restores as a user that only has privileges on the
	// test database, as on a typical shared host, rather than as root.
	restrictedRestore bool
}

func TestSaveLoadRoundTrip(t *testing.T) {
	cfg := loadDBConfig(t)

	for _, tc := range []struct {
		name  string
		flags []string
	}{
		{"gzip", nil},
		{"zstd", []string{"--zstd"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runScenario(t, cfg, scenario{seeds: []string{"seed.sql"}, assets: true, saveFlags: tc.flags})
		})
	}
}

// TestRoundTripTypes checks each group of column types separately, so a
// failure points at the type that does not survive a backup and restore.
func TestRoundTripTypes(t *testing.T) {
	cfg := loadDBConfig(t)

	for _, name := range []string{"numeric", "temporal", "text_binary", "bit", "json", "spatial"} {
		t.Run(name, func(t *testing.T) {
			runScenario(t, cfg, scenario{seeds: []string{"types_" + name + ".sql"}})
		})
	}
}

// TestRoundTripLargeData covers many rows (split across INSERT statements) and
// single rows larger than the dumper's packet size.
func TestRoundTripLargeData(t *testing.T) {
	runScenario(t, loadDBConfig(t), scenario{seeds: []string{"large.sql"}})
}

// TestRoundTripForeignKeys covers foreign keys, including a child table that
// sorts before its parent and a circular, self-referencing chain. These restore
// as a user with privileges on the test database only, as on a typical host.
func TestRoundTripForeignKeys(t *testing.T) {
	cfg := loadDBConfig(t)

	for _, name := range []string{"foreign_keys", "foreign_keys_circular"} {
		t.Run(name, func(t *testing.T) {
			runScenario(t, cfg, scenario{seeds: []string{name + ".sql"}, restrictedRestore: true})
		})
	}
}

// dbNameFor derives a database name unique to the (sub)test.
func dbNameFor(t *testing.T, cfg dbConfig) string {
	name := strings.ToLower(strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()))
	name = strings.TrimPrefix(name, "test")

	return cfg.Name + name
}

func writeEnv(t *testing.T, site string, cfg dbConfig, user, pass, dbName string) {
	t.Helper()

	env := fmt.Sprintf(`SS_ENVIRONMENT_TYPE="dev"
SS_DATABASE_CLASS="MySQLPDODatabase"
SS_DATABASE_SERVER=%q
SS_DATABASE_PORT=%q
SS_DATABASE_USERNAME=%q
SS_DATABASE_PASSWORD=%q
SS_DATABASE_NAME=%q
`, cfg.Host, cfg.Port, user, pass, dbName)
	require.NoError(t, os.WriteFile(filepath.Join(site, ".env"), []byte(env), 0o600))
}

func runScenario(t *testing.T, cfg dbConfig, sc scenario) {
	t.Helper()

	dbName := dbNameFor(t, cfg)
	admin := cfg.open(t, "")

	dropDB := func() {
		_, err := admin.Exec("DROP DATABASE IF EXISTS `" + dbName + "`")
		require.NoError(t, err)
	}

	dropDB()
	t.Cleanup(dropDB)

	// Seed the database.
	_, err := admin.Exec("CREATE DATABASE `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	require.NoError(t, err)

	db := cfg.open(t, dbName)

	for _, seed := range sc.seeds {
		b, err := seeds.ReadFile("testdata/" + seed)
		require.NoError(t, err)

		_, err = db.Exec(string(b))
		require.NoError(t, err, "seeding %s", seed)
	}

	// The credentials used for the restore.
	restoreUser, restorePass := cfg.User, cfg.Pass

	if sc.restrictedRestore {
		restoreUser, restorePass = "ssbak_e2e_user", "e2e-password"

		for _, q := range []string{
			fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", restoreUser, restorePass),
			fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", dbName, restoreUser),
		} {
			_, err := admin.Exec(q)
			require.NoError(t, err)
		}

		t.Cleanup(func() { _, _ = admin.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", restoreUser)) })
	}

	// Create the site: a .env adjacent to public/, and optionally public/assets.
	site := t.TempDir()
	assets := filepath.Join(site, "public", "assets")
	writeEnv(t, site, cfg, cfg.User, cfg.Pass, dbName)

	dbBefore := fingerprintDB(t, db, dbName)
	require.Greater(t, strings.Count(dbBefore, "\n"), 10, "database fingerprint looks empty")

	var assetsBefore string

	flags := []string{}
	if sc.assets {
		writeAssets(t, assets)

		assetsBefore = fingerprintAssets(t, assets)
		require.GreaterOrEqual(t, strings.Count(assetsBefore, "file "), 10, "assets fingerprint looks empty")
	} else {
		flags = append(flags, "--db")
	}

	// Save.
	sspak := filepath.Join(t.TempDir(), "backup.sspak")
	ssbak(t, append(append(append([]string{"save", "--verbose"}, flags...), sc.saveFlags...), site, sspak)...)

	ext := "gz"
	if slices.Contains(sc.saveFlags, "--zstd") {
		ext = "zst"
	}

	names := tarNames(t, sspak)
	assert.Contains(t, names, "database.sql."+ext)

	if sc.assets {
		assert.Contains(t, names, "assets.tar."+ext)
	} else {
		assert.NotContains(t, names, "assets.tar."+ext)
	}

	// Destroy the database and assets, and restore as the restore user.
	dropDB()

	if sc.assets {
		require.NoError(t, os.RemoveAll(assets))
	}

	writeEnv(t, site, cfg, restoreUser, restorePass, dbName)
	ssbak(t, append([]string{"load", "--verbose"}, append(flags, sspak, site)...)...)

	db = cfg.open(t, dbName)
	assert.Equal(t, dbBefore, fingerprintDB(t, db, dbName), "restored database differs from the original")

	if sc.assets {
		assert.Equal(t, assetsBefore, fingerprintAssets(t, assets), "restored assets differ from the original")
	}

	// Restoring over existing data must replace it, not merge with it.
	_, err = db.Exec("CREATE TABLE ssbak_stale_table (id INT)")
	require.NoError(t, err)

	if sc.assets {
		require.NoError(t, os.WriteFile(filepath.Join(assets, "stale.txt"), []byte("stale"), 0o600))
	}

	ssbak(t, append([]string{"load", "--verbose", "--drop-db"}, append(flags, sspak, site)...)...)

	db = cfg.open(t, dbName)
	assert.Equal(t, dbBefore, fingerprintDB(t, db, dbName), "database differs after restoring over existing data")

	if sc.assets {
		assert.Equal(t, assetsBefore, fingerprintAssets(t, assets), "assets differ after restoring over existing data")
		assert.NoFileExists(t, filepath.Join(assets, "stale.txt"))
	}
}

// asset describes a file to create under the assets directory.
type asset struct {
	path    string
	content []byte
	mode    os.FileMode
}

// writeAssets populates dir with a small but varied set of files and gives
// every file and directory a fixed modification time.
func writeAssets(t *testing.T, dir string) {
	t.Helper()

	random := make([]byte, 64*1024)
	_, _ = rand.New(rand.NewSource(1)).Read(random) //nolint:gosec // deterministic test data

	files := []asset{
		{"hello.txt", []byte("Hello, world!\n"), 0o755},
		{"empty.txt", nil, 0o644},
		{"Uploads/index.html", []byte("<html><body>Hi</body></html>\n"), 0o644},
		{"Uploads/nested/deeper/file.txt", []byte("nested file\n"), 0o644},
		{"Uploads/nested/file with spaces.txt", []byte("file with spaces\n"), 0o644},
		{"Uploads/nested/ünïcödé-日本語.txt", []byte("unicode name\n"), 0o644},
		{".protected/.htaccess", []byte("hidden\n"), 0o644},
		{"Uploads/binary.bin", []byte{0, 1, 2, 0xff, 0xfe, 0xfd, 0, 0, 'b', 'i', 'n', 0}, 0o644},
		{"Uploads/image.png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), 0o644},
		{"Uploads/random-64k.bin", random, 0o644},
		{"Uploads/_resampled/ResizedImageabc123/image.png", []byte("resampled\n"), 0o644},
	}

	require.NoError(t, os.MkdirAll(dir, 0o755))

	base := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)

	for i, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f.path))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, f.content, f.mode))
		require.NoError(t, os.Chmod(p, f.mode)) // not subject to the umask
		mtime := base.Add(time.Duration(i) * time.Hour)
		require.NoError(t, os.Chtimes(p, mtime, mtime))
	}

	// Set directory times last, deepest first, as writing files alters them.
	var dirs []string

	require.NoError(t, filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			dirs = append(dirs, p)
		}

		return err
	}))

	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })

	for i, d := range dirs {
		mtime := base.Add(-time.Duration(i+1) * time.Hour)
		require.NoError(t, os.Chtimes(d, mtime, mtime))
	}
}

// fingerprintAssets describes every file and directory under dir (relative
// path, mode, size, content hash and modification time) for comparison.
func fingerprintAssets(t *testing.T, dir string) string {
	t.Helper()

	var lines []string

	require.NoError(t, filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(dir, p)

		info, err := os.Lstat(p)
		if err != nil {
			return err
		}

		if d.IsDir() {
			lines = append(lines, fmt.Sprintf("dir  %s mode=%o mtime=%d", rel, info.Mode().Perm(), info.ModTime().Unix()))

			return nil
		}

		b, err := os.ReadFile(p) //nolint:gosec // test paths
		if err != nil {
			return err
		}

		lines = append(lines, fmt.Sprintf("file %s mode=%o size=%d sha256=%x mtime=%d",
			rel, info.Mode().Perm(), info.Size(), sha256.Sum256(b), info.ModTime().Unix()))

		return nil
	}))

	return strings.Join(lines, "\n") + "\n"
}

func tarNames(t *testing.T, path string) []string {
	t.Helper()

	f, err := os.Open(path) //nolint:gosec // test paths
	require.NoError(t, err)

	defer func() { _ = f.Close() }()

	var names []string

	tr := tar.NewReader(f)

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)

		names = append(names, h.Name)
	}

	return names
}

// queryRows runs a query and returns every cell as a string, with NULL shown
// as "NULL".
func queryRows(t *testing.T, db *sql.DB, query string) [][]string {
	t.Helper()

	rows, err := db.Query(query)
	require.NoError(t, err, query)

	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	require.NoError(t, err)

	var out [][]string

	for rows.Next() {
		vals := make([]sql.RawBytes, len(cols))
		ptrs := make([]any, len(cols))

		for i := range vals {
			ptrs[i] = &vals[i]
		}

		require.NoError(t, rows.Scan(ptrs...))

		row := make([]string, len(cols))
		for i, v := range vals {
			row[i] = formatCell(v)
		}

		out = append(out, row)
	}

	require.NoError(t, rows.Err())

	return out
}

// formatCell renders raw column bytes unambiguously: small values are quoted
// (so binary bytes are escaped), large values are shown as a length and hash.
func formatCell(v []byte) string {
	switch {
	case v == nil:
		return "NULL"
	case len(v) <= 64:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprintf("<%d bytes sha256:%x>", len(v), sha256.Sum256(v))
	}
}

// fingerprintDB returns a deterministic description of the whole database:
// columns, indexes, keys, views, table properties and every row of every table
// and view. Tables are discovered from information_schema, so new test fixtures
// are compared without any extra code.
func fingerprintDB(t *testing.T, db *sql.DB, name string) string {
	t.Helper()

	var b strings.Builder

	section := func(title, query string) [][]string {
		rows := queryRows(t, db, query)

		fmt.Fprintf(&b, "### %s\n", title)

		for _, r := range rows {
			b.WriteString(strings.Join(r, "\t"))
			b.WriteString("\n")
		}

		return rows
	}

	// Columns are read from information_schema rather than SHOW CREATE TABLE,
	// as the latter varies cosmetically between servers.
	section("columns", fmt.Sprintf(`SELECT table_name, column_name, ordinal_position, column_type, is_nullable,
		column_default, character_set_name, collation_name, column_key, extra, generation_expression
		FROM information_schema.columns WHERE table_schema = '%s'
		ORDER BY table_name, ordinal_position`, name))

	section("indexes", fmt.Sprintf(`SELECT table_name, index_name, seq_in_index, column_name, non_unique
		FROM information_schema.statistics WHERE table_schema = '%s'
		ORDER BY table_name, index_name, seq_in_index`, name))

	section("foreign keys", fmt.Sprintf(`SELECT constraint_name, table_name, referenced_table_name, update_rule, delete_rule
		FROM information_schema.referential_constraints WHERE constraint_schema = '%s'
		ORDER BY constraint_name`, name))

	// The definer is deliberately excluded: it changes when restoring as a
	// different user.
	section("views", fmt.Sprintf(`SELECT table_name, view_definition, check_option, is_updatable, security_type
		FROM information_schema.views WHERE table_schema = '%s' ORDER BY table_name`, name))

	// A never-used table reports a NULL auto_increment but 1 once restored.
	section("tables", fmt.Sprintf(`SELECT table_name, engine, table_collation,
		GREATEST(COALESCE(auto_increment, 1), 1)
		FROM information_schema.tables WHERE table_schema = '%s' AND table_type = 'BASE TABLE'
		ORDER BY table_name`, name))

	// Data for every table and view.
	tables := queryRows(t, db, fmt.Sprintf(`SELECT table_name FROM information_schema.tables
		WHERE table_schema = '%s' AND table_type IN ('BASE TABLE', 'VIEW') ORDER BY table_name`, name))

	for _, tbl := range tables {
		table := strings.Trim(tbl[0], `"`)

		cols := queryRows(t, db, fmt.Sprintf(`SELECT column_name FROM information_schema.columns
			WHERE table_schema = '%s' AND table_name = '%s' ORDER BY ordinal_position`, name, table))

		names := make([]string, len(cols))
		positions := make([]string, len(cols))

		for i, c := range cols {
			names[i] = "`" + strings.Trim(c[0], `"`) + "`"
			positions[i] = fmt.Sprint(i + 1)
		}

		// Order by the primary key where there is one (some types, such as
		// geometries, cannot be sorted), otherwise by every column.
		order := strings.Join(positions, ", ")

		pk := queryRows(t, db, fmt.Sprintf(`SELECT column_name FROM information_schema.key_column_usage
			WHERE table_schema = '%s' AND table_name = '%s' AND constraint_name = 'PRIMARY'
			ORDER BY ordinal_position`, name, table))
		if len(pk) > 0 {
			keys := make([]string, len(pk))
			for i, k := range pk {
				keys[i] = "`" + strings.Trim(k[0], `"`) + "`"
			}

			order = strings.Join(keys, ", ")
		}

		fmt.Fprintf(&b, "### data: %s\n", table)

		rows := queryRows(t, db, fmt.Sprintf("SELECT %s FROM `%s` ORDER BY %s",
			strings.Join(names, ", "), table, order))

		for _, r := range rows {
			cells := make([]string, len(r))
			for i := range r {
				cells[i] = names[i] + "=" + r[i]
			}

			b.WriteString(strings.Join(cells, "\t"))
			b.WriteString("\n")
		}
	}

	return b.String()
}
