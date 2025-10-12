DROP TABLE IF EXISTS `test`;
CREATE TABLE `test` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `index` int NOT NULL DEFAULT '0',
  `int_p` int DEFAULT NULL,
  `bool` tinyint NOT NULL DEFAULT '0',
  `bool_p` tinyint DEFAULT NULL,
  `str` varchar(255) NOT NULL DEFAULT '',
  `str_p` varchar(255) DEFAULT NULL,
  `timestamp` timestamp NOT NULL,
  `timestamp_p` datetime DEFAULT NULL,
  `datetime` datetime NOT NULL,
  `datetime_p` datetime DEFAULT NULL,
  `decimal` decimal(10,2) NOT NULL,
  `decimal_p` decimal(10,2) DEFAULT NULL,
  `int_slice` varchar(255) NOT NULL,
  `int_slice_p` varchar(255) DEFAULT NULL,
  `struct` varchar(255) NOT NULL,
  `struct_p` varchar(255) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;