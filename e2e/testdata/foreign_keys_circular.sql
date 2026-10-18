-- A self-referencing table holding a circular chain (1 -> 3 -> 2 -> 1), so rows
-- reference ids that are only inserted later.
CREATE TABLE tree (
  id INT NOT NULL,
  parent_id INT DEFAULT NULL,
  PRIMARY KEY (id),
  KEY parent_id (parent_id),
  CONSTRAINT fk_tree_self FOREIGN KEY (parent_id) REFERENCES tree (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET FOREIGN_KEY_CHECKS = 0;
INSERT INTO tree VALUES (1, 3), (2, 1), (3, 2), (4, NULL);
SET FOREIGN_KEY_CHECKS = 1;
