CREATE TABLE bench_users (
  id BIGINT NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  alias VARCHAR(255) DEFAULT NULL,
  age INT NOT NULL,
  age_p INT DEFAULT NULL,
  active TINYINT(1) NOT NULL DEFAULT 0,
  active_p TINYINT(1) DEFAULT NULL,
  email VARCHAR(255) NOT NULL,
  tags JSON NOT NULL,
  meta JSON NOT NULL,
  profile JSON NOT NULL,
  contacts JSON NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_bench_users_email (email),
  KEY idx_bench_users_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
