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
