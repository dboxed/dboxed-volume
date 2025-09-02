-- +goose Up
-- create "user" table
CREATE TABLE `user` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (current_timestamp),
  `name` text NOT NULL,
  `email` text NULL,
  `avatar` text NULL,
  PRIMARY KEY (`id`)
);
-- create "repository" table
CREATE TABLE `repository` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NOT NULL DEFAULT (current_timestamp),
  `deleted_at` datetime NULL,
  `finalizers` text NOT NULL DEFAULT '{}',
  `name` text NOT NULL,
  `uuid` text NOT NULL
);
-- create index "repository_uuid" to table: "repository"
CREATE UNIQUE INDEX `repository_uuid` ON `repository` (`uuid`);
-- create index "repository_name" to table: "repository"
CREATE UNIQUE INDEX `repository_name` ON `repository` (`name`);
-- create "repository_storage_s3" table
CREATE TABLE `repository_storage_s3` (
  `id` bigint NULL,
  `endpoint` text NOT NULL,
  `region` text NULL,
  `bucket` text NOT NULL,
  `access_key_id` text NOT NULL,
  `secret_access_key` text NOT NULL,
  `prefix` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`id`) REFERENCES `repository` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create "repository_backup_rustic" table
CREATE TABLE `repository_backup_rustic` (
  `id` bigint NULL,
  `password` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`id`) REFERENCES `repository` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create "volume" table
CREATE TABLE `volume` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NOT NULL DEFAULT (current_timestamp),
  `deleted_at` datetime NULL,
  `finalizers` text NOT NULL DEFAULT '{}',
  `repository_id` bigint NOT NULL,
  `uuid` text NOT NULL,
  `name` text NOT NULL,
  `fs_size` bigint NOT NULL,
  `fs_type` text NOT NULL,
  `lock_id` text NULL,
  `lock_time` bigint NULL,
  CONSTRAINT `0` FOREIGN KEY (`repository_id`) REFERENCES `repository` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT
);
-- create index "volume_uuid" to table: "volume"
CREATE UNIQUE INDEX `volume_uuid` ON `volume` (`uuid`);
-- create index "volume_repository_id_uuid" to table: "volume"
CREATE UNIQUE INDEX `volume_repository_id_uuid` ON `volume` (`repository_id`, `uuid`);
-- create index "volume_repository_id_name" to table: "volume"
CREATE UNIQUE INDEX `volume_repository_id_name` ON `volume` (`repository_id`, `name`);

-- +goose Down
-- reverse: create index "volume_repository_id_name" to table: "volume"
DROP INDEX `volume_repository_id_name`;
-- reverse: create index "volume_repository_id_uuid" to table: "volume"
DROP INDEX `volume_repository_id_uuid`;
-- reverse: create index "volume_uuid" to table: "volume"
DROP INDEX `volume_uuid`;
-- reverse: create "volume" table
DROP TABLE `volume`;
-- reverse: create "repository_backup_rustic" table
DROP TABLE `repository_backup_rustic`;
-- reverse: create "repository_storage_s3" table
DROP TABLE `repository_storage_s3`;
-- reverse: create index "repository_name" to table: "repository"
DROP INDEX `repository_name`;
-- reverse: create index "repository_uuid" to table: "repository"
DROP INDEX `repository_uuid`;
-- reverse: create "repository" table
DROP TABLE `repository`;
-- reverse: create "user" table
DROP TABLE `user`;
