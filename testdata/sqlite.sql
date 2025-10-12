DROP TABLE IF EXISTS test;
CREATE TABLE test (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  index INTEGER NOT NULL DEFAULT 0,
  int_p INTEGER,
  bool BOOLEAN NOT NULL DEFAULT 0,
  bool_p BOOLEAN,
  str TEXT NOT NULL DEFAULT '',
  str_p TEXT,
  timestamp DATETIME NOT NULL,
  timestamp_p DATETIME,
  datetime DATETIME NOT NULL,
  datetime_p DATETIME,
  decimal DECIMAL(10,2) NOT NULL,
  decimal_p DECIMAL(10,2),
  int_slice TEXT NOT NULL,
  int_slice_p TEXT,
  struct TEXT NOT NULL,
  struct_p TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);