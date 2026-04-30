DROP TABLE IF EXISTS test;
CREATE TABLE test (
  id BIGINT IDENTITY(1,1) PRIMARY KEY,
  [index] INT NOT NULL DEFAULT 0,
  int_p INT,
  bool BIT NOT NULL DEFAULT 0,
  bool_p BIT,
  str VARCHAR(255) NOT NULL DEFAULT '',
  str_p VARCHAR(255),
  timestamp DATETIME NOT NULL,
  timestamp_p DATETIME,
  datetime DATETIME NOT NULL,
  datetime_p DATETIME,
  decimal DECIMAL(10,2) NOT NULL,
  decimal_p DECIMAL(10,2),
  int_slice VARCHAR(255) NOT NULL,
  int_slice_p VARCHAR(255),
  struct VARCHAR(255) NOT NULL,
  struct_p VARCHAR(255),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
