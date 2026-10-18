-- Many small rows: the dump is split into several INSERT statements as it
-- exceeds the dumper's packet size.
CREATE TABLE digits (d INT);
INSERT INTO digits VALUES (0),(1),(2),(3),(4),(5),(6),(7),(8),(9);

CREATE TABLE many_rows (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  payload VARCHAR(255) NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO many_rows (payload)
  SELECT CONCAT('row-', a.d + b.d * 10 + c.d * 100 + e.d * 1000, '-', REPEAT('x', 100))
  FROM digits a, digits b, digits c, digits e;

-- Single rows larger than the dumper's packet size (about 1.2MB each).
CREATE TABLE big_values (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  big_blob LONGBLOB,
  big_text LONGTEXT,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO big_values (big_blob, big_text) VALUES
  (REPEAT(UNHEX('00FF10'), 400000), REPEAT('é', 600000)),
  (REPEAT(UNHEX('27225C'), 400000), REPEAT('quote'' and \\ ', 100000));

DROP TABLE digits;
