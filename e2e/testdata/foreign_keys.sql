-- Names are chosen so that the child sorts alphabetically BEFORE its parent,
-- so a restore only works if foreign key checks are disabled.
CREATE TABLE z_parent (
  id INT NOT NULL,
  name VARCHAR(50) NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE a_child (
  id INT NOT NULL,
  parent_id INT NOT NULL,
  PRIMARY KEY (id),
  KEY parent_id (parent_id),
  CONSTRAINT fk_child_parent FOREIGN KEY (parent_id) REFERENCES z_parent (id) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO z_parent VALUES (1, 'first'), (2, 'second'), (3, 'third');
INSERT INTO a_child VALUES (10, 1), (11, 1), (12, 3);
