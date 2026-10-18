CREATE TABLE t_spatial (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  c_geom GEOMETRY DEFAULT NULL,
  c_point POINT DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO t_spatial (c_geom, c_point) VALUES
  (ST_GeomFromText('POINT(1 2)'), ST_GeomFromText('POINT(3 4)')),
  (ST_GeomFromText('LINESTRING(0 0, 10 10, 20 25)'), ST_GeomFromText('POINT(-180 -90)')),
  (ST_GeomFromText('POLYGON((0 0, 0 10, 10 10, 10 0, 0 0))'), ST_GeomFromText('POINT(180 90)')),
  (NULL, NULL);
