#!/usr/bin/env bash
#
# End-to-end test for ssbak: save a site to an .sspak, destroy it, restore it,
# then verify the database and assets match the originals exactly.
#
# Requires: a running MySQL/MariaDB server, the `mysql` client, and an ssbak binary.
#
# Environment (defaults match the GitHub workflow):
#   SSBAK_BIN        path to the ssbak binary          (default: ./ssbak)
#   TEST_DB_HOST     database host                     (default: 127.0.0.1)
#   TEST_DB_PORT     database port                     (default: 3306)
#   TEST_DB_USER     database user                     (default: root)
#   TEST_DB_PASS     database password                 (default: root)
#   TEST_DB_NAME     database name                     (default: ssbak_e2e)
#
# Usage: e2e/e2e.sh [save-flags...]   e.g. e2e/e2e.sh --zstd

set -euo pipefail

SSBAK_BIN="$(realpath "${SSBAK_BIN:-./ssbak}")"
DB_HOST="${TEST_DB_HOST:-127.0.0.1}"
DB_PORT="${TEST_DB_PORT:-3306}"
DB_USER="${TEST_DB_USER:-root}"
DB_PASS="${TEST_DB_PASS:-root}"
DB_NAME="${TEST_DB_NAME:-ssbak_e2e}"
SAVE_FLAGS=("$@")

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

SITE="$WORK/site"
ASSETS="$SITE/public/assets"
SSPAK="$WORK/backup.sspak"

# mysql <db> reads SQL from stdin. The password is passed via the environment to
# keep it out of the process list.
mysql_cmd() {
  MYSQL_PWD="$DB_PASS" mysql --protocol=tcp -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" \
    --default-character-set=utf8mb4 --batch --raw "$@"
}

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

step() {
  echo
  echo "==> $*"
}

# fingerprint_db writes a deterministic description of every table (columns,
# indexes and data, with binary columns hex-encoded) so before/after can be
# compared with diff. Columns are read from information_schema rather than
# SHOW CREATE TABLE, as the latter varies cosmetically between servers/clients.
fingerprint_db() {
  local out="$1"
  {
    echo "### columns"
    mysql_cmd -N -e "SELECT table_name, column_name, ordinal_position, column_type, is_nullable, column_default, character_set_name, collation_name, column_key, extra FROM information_schema.columns WHERE table_schema = '$DB_NAME' ORDER BY table_name, ordinal_position"
    echo "### indexes"
    mysql_cmd -N -e "SELECT table_name, index_name, seq_in_index, column_name, non_unique FROM information_schema.statistics WHERE table_schema = '$DB_NAME' ORDER BY table_name, index_name, seq_in_index"
    echo "### tables"
    mysql_cmd -N -e "SELECT table_name, engine, table_collation, GREATEST(COALESCE(auto_increment, 1), 1) FROM information_schema.tables WHERE table_schema = '$DB_NAME' ORDER BY table_name"
    echo "### users"
    mysql_cmd -N -e "SELECT id, name, email, bio, HEX(CAST(bio AS BINARY)), active, balance, created_at FROM users ORDER BY id" "$DB_NAME"
    echo "### files"
    # Small blobs are shown as hex for readable diffs; every blob is also hashed.
    mysql_cmd -N -e "SELECT id, filename, size, content IS NULL, LENGTH(content), IF(LENGTH(content) <= 256, HEX(content), '(large)'), SHA2(content, 256) FROM files ORDER BY id" "$DB_NAME"
    echo "### raw_binary"
    mysql_cmd -N -e "SELECT id, HEX(fixed), HEX(variable), LENGTH(variable) FROM raw_binary ORDER BY id" "$DB_NAME"
    echo "### settings"
    mysql_cmd -N -e "SELECT \`key\`, value IS NULL, HEX(CAST(value AS BINARY)) FROM settings ORDER BY \`key\`" "$DB_NAME"
    echo "### counts"
    mysql_cmd -N -e "SELECT 'users', COUNT(*) FROM users UNION ALL SELECT 'files', COUNT(*) FROM files UNION ALL SELECT 'raw_binary', COUNT(*) FROM raw_binary UNION ALL SELECT 'settings', COUNT(*) FROM settings UNION ALL SELECT 'empty_table', COUNT(*) FROM empty_table" "$DB_NAME"
  } >"$out"
}

# fingerprint_assets writes the relative path, size and sha256 of every file,
# plus every directory, so before/after can be compared with diff.
fingerprint_assets() {
  local out="$1"
  (
    cd "$ASSETS"
    {
      find . -type d | sort | sed 's/^/dir  /'
      find . -type f -print0 | sort -z | xargs -0 sha256sum | sed 's/^/file /'
      find . -type f -print0 | sort -z | xargs -0 stat -c 'size %s %n'
    }
  ) >"$out"
}

step "Creating database '$DB_NAME'"
mysql_cmd -e "DROP DATABASE IF EXISTS \`$DB_NAME\`; CREATE DATABASE \`$DB_NAME\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"

mysql_cmd "$DB_NAME" <<'SQL'
CREATE TABLE users (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  email VARCHAR(255) DEFAULT NULL,
  bio TEXT,
  active TINYINT(1) NOT NULL DEFAULT 1,
  balance DECIMAL(10,2) DEFAULT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE files (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  filename VARCHAR(255) NOT NULL,
  size INT UNSIGNED NOT NULL,
  content LONGBLOB,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE settings (
  `key` VARCHAR(50) NOT NULL,
  value TEXT,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- BINARY/VARBINARY columns (as opposed to BLOBs), e.g. hashes and tokens.
CREATE TABLE raw_binary (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  fixed BINARY(16) NOT NULL,
  variable VARBINARY(64) DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE empty_table (
  id INT NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

-- Regular records, including NULLs, quotes, backslashes, newlines and unicode.
INSERT INTO users (name, email, bio, active, balance, created_at) VALUES
  ('Alice', 'alice@example.com', 'Plain text bio', 1, 1234.56, '2024-01-02 03:04:05'),
  ('Bob O''Brien', NULL, NULL, 0, NULL, '2000-02-29 23:59:59'),
  ('Back\\slash "quoted"', 'bs@example.com', 'Line one\nLine two\ttabbed\r\nLine three', 1, -0.01, '2024-12-31 00:00:00'),
  ('Ünïcödé 日本語', 'unicode@example.com', 'Emoji 🎉🚀 and ñ and ß', 1, 99999999.99, '2025-06-15 12:30:00'),
  ('Semi;colon', 'semi@example.com', 'DROP TABLE users; -- not an injection', 1, 0.00, '2025-01-01 00:00:00');

-- Binary data: every byte value 0x00-0xFF, NUL bytes, high bytes, invalid UTF-8,
-- a PNG header, an empty blob and a NULL blob.
INSERT INTO files (filename, size, content) VALUES
  ('all-bytes.bin', 256, UNHEX('000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F202122232425262728292A2B2C2D2E2F303132333435363738393A3B3C3D3E3F404142434445464748494A4B4C4D4E4F505152535455565758595A5B5C5D5E5F606162636465666768696A6B6C6D6E6F707172737475767778797A7B7C7D7E7F808182838485868788898A8B8C8D8E8F909192939495969798999A9B9C9D9E9FA0A1A2A3A4A5A6A7A8A9AAABACADAEAFB0B1B2B3B4B5B6B7B8B9BABBBCBDBEBFC0C1C2C3C4C5C6C7C8C9CACBCCCDCECFD0D1D2D3D4D5D6D7D8D9DADBDCDDDEDFE0E1E2E3E4E5E6E7E8E9EAEBECEDEEEFF0F1F2F3F4F5F6F7F8F9FAFBFCFDFEFF')),
  ('nul-bytes.bin', 8, UNHEX('0000000000000000')),
  ('invalid-utf8.bin', 7, UNHEX('C328A0A1E28228')),
  ('quotes-and-escapes.bin', 8, UNHEX('27225C5C0A0D1A00')),
  ('png-header.png', 16, UNHEX('89504E470D0A1A0A0000000D49484452')),
  ('empty.bin', 0, ''),
  ('null.bin', 0, NULL);

-- A larger pseudo-random blob (4096 bytes) built from repeated hashes.
INSERT INTO files (filename, size, content)
  SELECT 'random-4k.bin', 4096,
    UNHEX(CONCAT(
      SHA2('a', 512), SHA2('b', 512), SHA2('c', 512), SHA2('d', 512),
      SHA2('e', 512), SHA2('f', 512), SHA2('g', 512), SHA2('h', 512),
      SHA2('i', 512), SHA2('j', 512), SHA2('k', 512), SHA2('l', 512),
      SHA2('m', 512), SHA2('n', 512), SHA2('o', 512), SHA2('p', 512),
      SHA2('q', 512), SHA2('r', 512), SHA2('s', 512), SHA2('t', 512),
      SHA2('u', 512), SHA2('v', 512), SHA2('w', 512), SHA2('x', 512),
      SHA2('y', 512), SHA2('z', 512), SHA2('A', 512), SHA2('B', 512),
      SHA2('C', 512), SHA2('D', 512), SHA2('E', 512), SHA2('F', 512),
      SHA2('G', 512), SHA2('H', 512), SHA2('I', 512), SHA2('J', 512),
      SHA2('K', 512), SHA2('L', 512), SHA2('M', 512), SHA2('N', 512),
      SHA2('O', 512), SHA2('P', 512), SHA2('Q', 512), SHA2('R', 512),
      SHA2('S', 512), SHA2('T', 512), SHA2('U', 512), SHA2('V', 512),
      SHA2('W', 512), SHA2('X', 512), SHA2('Y', 512), SHA2('Z', 512),
      SHA2('0', 512), SHA2('1', 512), SHA2('2', 512), SHA2('3', 512),
      SHA2('4', 512), SHA2('5', 512), SHA2('6', 512), SHA2('7', 512),
      SHA2('8', 512), SHA2('9', 512), SHA2('-', 512), SHA2('_', 512)
    ));

INSERT INTO raw_binary (fixed, variable) VALUES
  (UNHEX('00112233445566778899AABBCCDDEEFF'), UNHEX('FFFEFDFC00010203')),
  (UNHEX('000000000000000000000000000000FF'), NULL),
  (UNHEX('80818283848586878889C3288081A0A1'), UNHEX('27225C5C0A0D1A00'));

INSERT INTO settings (`key`, value) VALUES
  ('site_title', 'My Test Site'),
  ('motd', 'Hello, ''world''!'),
  ('binary_ish', CONCAT('a', CHAR(0), 'b')),
  ('empty', ''),
  ('nothing', NULL);
SQL

# The random blob must really be 4096 bytes, otherwise the test data is wrong.
[ "$(mysql_cmd -N -e "SELECT LENGTH(content) FROM files WHERE filename='random-4k.bin'" "$DB_NAME")" = "4096" ] ||
  fail "test data generation: random-4k.bin is not 4096 bytes"

step "Creating site files in $SITE"
mkdir -p "$ASSETS/Uploads/nested/deeper" "$ASSETS/Uploads/_resampled/ResizedImageabc123" "$ASSETS/.protected"
printf 'Hello, world!\n' >"$ASSETS/hello.txt"
printf '' >"$ASSETS/empty.txt"
printf '<html><body>Hi</body></html>\n' >"$ASSETS/Uploads/index.html"
printf 'nested file\n' >"$ASSETS/Uploads/nested/deeper/file.txt"
printf 'file with spaces\n' >"$ASSETS/Uploads/nested/file with spaces.txt"
printf 'unicode name\n' >"$ASSETS/Uploads/nested/ünïcödé-日本語.txt"
printf 'hidden\n' >"$ASSETS/.protected/.htaccess"
printf '\x00\x01\x02\xff\xfe\xfd\x00\x00binary\x00' >"$ASSETS/Uploads/binary.bin"
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR' >"$ASSETS/Uploads/image.png"
head -c 65536 /dev/urandom >"$ASSETS/Uploads/random-64k.bin"
printf 'resampled\n' >"$ASSETS/Uploads/_resampled/ResizedImageabc123/image.png"
chmod 0755 "$ASSETS/hello.txt"

# .env sits at the base of the site, adjacent to public/
cat >"$SITE/.env" <<ENV
SS_ENVIRONMENT_TYPE="dev"
SS_DATABASE_CLASS="MySQLPDODatabase"
SS_DATABASE_SERVER="$DB_HOST"
SS_DATABASE_PORT="$DB_PORT"
SS_DATABASE_USERNAME="$DB_USER"
SS_DATABASE_PASSWORD="$DB_PASS"
SS_DATABASE_NAME="$DB_NAME"
ENV

# Make sure nothing from the calling environment leaks past the .env file.
unset SS_DATABASE_SERVER SS_DATABASE_PORT SS_DATABASE_USERNAME SS_DATABASE_PASSWORD \
  SS_DATABASE_NAME SS_DATABASE_CLASS SS_DATABASE_PREFIX SS_DATABASE_SUFFIX \
  SS_DATABASE_CHOOSE_NAME SS_IGNORE_DOT_ENV

step "Recording original state"
fingerprint_db "$WORK/db.before"
fingerprint_assets "$WORK/assets.before"
[ "$(grep -c . "$WORK/db.before")" -gt 40 ] || fail "database fingerprint looks empty"
[ "$(grep -c '^file ' "$WORK/assets.before")" -ge 10 ] || fail "assets fingerprint looks empty"
stat -c '%a' "$ASSETS/hello.txt" >"$WORK/mode.before"

step "ssbak save ${SAVE_FLAGS[*]:-}"
"$SSBAK_BIN" save --verbose "${SAVE_FLAGS[@]}" "$SITE" "$SSPAK"
[ -s "$SSPAK" ] || fail "sspak file was not created"

step "Checking sspak contents"
tar -tf "$SSPAK" | tee "$WORK/sspak.list"
grep -q '^database\.sql\.\(gz\|zst\)$' "$WORK/sspak.list" || fail "sspak is missing the database dump"
grep -q '^assets\.tar\.\(gz\|zst\)$' "$WORK/sspak.list" || fail "sspak is missing the assets archive"

step "Destroying database and assets"
mysql_cmd -e "DROP DATABASE \`$DB_NAME\`"
rm -rf "$ASSETS"
[ ! -e "$ASSETS" ] || fail "assets were not deleted"
if mysql_cmd -N -e "SHOW DATABASES LIKE '$DB_NAME'" | grep -q .; then
  fail "database was not dropped"
fi

step "ssbak load"
"$SSBAK_BIN" load --verbose "$SSPAK" "$SITE"

step "Verifying restored database"
fingerprint_db "$WORK/db.after"
diff -au "$WORK/db.before" "$WORK/db.after" || fail "database does not match the original"
echo "Database matches ($(grep -c . "$WORK/db.after") fingerprint lines)"

step "Verifying restored assets"
[ -d "$ASSETS" ] || fail "assets directory was not restored"
fingerprint_assets "$WORK/assets.after"
diff -au "$WORK/assets.before" "$WORK/assets.after" || fail "assets do not match the original"
[ "$(stat -c '%a' "$ASSETS/hello.txt")" = "$(cat "$WORK/mode.before")" ] || fail "file mode was not preserved"
echo "Assets match ($(grep -c '^file ' "$WORK/assets.after") files)"

step "Verifying the restore really replaces data (second load over existing data)"
mysql_cmd -e "INSERT INTO settings (\`key\`, value) VALUES ('stale', 'should be removed')" "$DB_NAME"
echo "stale" >"$ASSETS/stale.txt"
"$SSBAK_BIN" load --verbose --drop-db "$SSPAK" "$SITE"
fingerprint_db "$WORK/db.after2"
diff -au "$WORK/db.before" "$WORK/db.after2" || fail "database does not match the original after reload"
[ ! -e "$ASSETS/stale.txt" ] || fail "stale asset survived the restore"
fingerprint_assets "$WORK/assets.after2"
diff -au "$WORK/assets.before" "$WORK/assets.after2" || fail "assets do not match the original after reload"

echo
echo "PASS: ssbak save/load round trip (${SAVE_FLAGS[*]:-default compression})"
