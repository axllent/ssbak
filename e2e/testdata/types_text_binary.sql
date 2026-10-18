CREATE TABLE t_text_binary (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  c_char CHAR(10) DEFAULT NULL,
  c_varchar VARCHAR(255) DEFAULT NULL,
  c_tinytext TINYTEXT,
  c_text TEXT,
  c_mediumtext MEDIUMTEXT,
  c_longtext LONGTEXT,
  c_tinyblob TINYBLOB,
  c_blob BLOB,
  c_mediumblob MEDIUMBLOB,
  c_longblob LONGBLOB,
  c_binary BINARY(4) DEFAULT NULL,
  c_varbinary VARBINARY(16) DEFAULT NULL,
  c_enum ENUM('a','b c','d''e') DEFAULT NULL,
  c_set SET('x','y','z') DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO t_text_binary
  (c_char, c_varchar, c_tinytext, c_text, c_mediumtext, c_longtext, c_tinyblob, c_blob, c_mediumblob, c_longblob, c_binary, c_varbinary, c_enum, c_set)
VALUES
  ('abc', 'Plain', 'tiny', 'text', 'medium', 'long',
   UNHEX('00'), UNHEX('00FF'), UNHEX('0001FEFF'), UNHEX('C328A0A1'), UNHEX('01020304'), UNHEX('FFFEFDFC00010203'), 'a', 'x'),
  ('trailing  ', 'Quotes '' " \\ and a newline
and a tab	end', 'ünï', '日本語 🎉', REPEAT('m', 70000), REPEAT('é', 70000),
   '', '', '', '', UNHEX('00000000'), '', 'd''e', 'x,y,z'),
  (NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL),
  ('', '', '', '', '', '', UNHEX('27225C5C0A0D1A00'), UNHEX(REPEAT('AB', 30000)), UNHEX(REPEAT('00', 70000)), UNHEX(REPEAT('FF', 70000)), UNHEX('80C3'), UNHEX('00'), 'b c', '');
