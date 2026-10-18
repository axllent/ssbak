CREATE TABLE t_json (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  c_json JSON DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO t_json (c_json) VALUES
  ('{"a": [1, 2, {"b": null}], "unicode": "日本語 🎉", "quote": "it''s \\"quoted\\"", "n": 1.5e10}'),
  ('"just a string"'),
  ('[]'),
  ('null'),
  (NULL);
