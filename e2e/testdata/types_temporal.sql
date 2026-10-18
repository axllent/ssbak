-- A relaxed sql_mode is needed to store zero dates, which old (Silverstripe 3
-- era) databases genuinely contain.
SET SESSION sql_mode = '';

CREATE TABLE t_temporal (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  c_year YEAR DEFAULT NULL,
  c_date DATE DEFAULT NULL,
  c_time TIME(3) DEFAULT NULL,
  c_datetime DATETIME(6) DEFAULT NULL,
  c_timestamp TIMESTAMP(6) NULL DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO t_temporal (c_year, c_date, c_time, c_datetime, c_timestamp) VALUES
  (1901, '1000-01-01', '-838:59:59.000', '1000-01-01 00:00:00.000000', '1970-01-02 00:00:01.000001'),
  (2155, '9999-12-31', '838:59:59.999', '9999-12-31 23:59:59.999999', '2038-01-19 03:14:07.999999'),
  (NULL, NULL, NULL, NULL, NULL),
  (NULL, '0000-00-00', '00:00:00.000', '0000-00-00 00:00:00.000000', '0000-00-00 00:00:00.000000'),
  (2024, '2024-02-29', '12:34:56.789', '2024-02-29 12:34:56.789012', '2024-02-29 12:34:56.789012');

SET SESSION sql_mode = DEFAULT;
