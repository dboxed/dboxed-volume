-- +goose Up
-- create "repository" table
CREATE TABLE "repository" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz NULL,
  "finalizers" text NOT NULL DEFAULT '{}',
  "name" text NOT NULL,
  "uuid" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "repository_name_key" UNIQUE ("name"),
  CONSTRAINT "repository_uuid_key" UNIQUE ("uuid")
);
-- create "user" table
CREATE TABLE "user" (
  "id" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "name" text NOT NULL,
  "email" text NULL,
  "avatar" text NULL,
  PRIMARY KEY ("id")
);
-- create "repository_backup_rustic" table
CREATE TABLE "repository_backup_rustic" (
  "id" bigint NOT NULL,
  "password" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "repository_backup_rustic_id_fkey" FOREIGN KEY ("id") REFERENCES "repository" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create "repository_storage_s3" table
CREATE TABLE "repository_storage_s3" (
  "id" bigint NOT NULL,
  "endpoint" text NOT NULL,
  "region" text NULL,
  "bucket" text NOT NULL,
  "access_key_id" text NOT NULL,
  "secret_access_key" text NOT NULL,
  "prefix" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "repository_storage_s3_id_fkey" FOREIGN KEY ("id") REFERENCES "repository" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create "volume" table
CREATE TABLE "volume" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamptz NULL,
  "finalizers" text NOT NULL DEFAULT '{}',
  "repository_id" bigint NOT NULL,
  "uuid" text NOT NULL,
  "name" text NOT NULL,
  "fs_size" bigint NOT NULL,
  "fs_type" text NOT NULL,
  "lock_id" text NULL,
  "lock_time" bigint NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "volume_repository_id_name_key" UNIQUE ("repository_id", "name"),
  CONSTRAINT "volume_repository_id_uuid_key" UNIQUE ("repository_id", "uuid"),
  CONSTRAINT "volume_uuid_key" UNIQUE ("uuid"),
  CONSTRAINT "volume_repository_id_fkey" FOREIGN KEY ("repository_id") REFERENCES "repository" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

-- +goose Down
-- reverse: create "volume" table
DROP TABLE "volume";
-- reverse: create "repository_storage_s3" table
DROP TABLE "repository_storage_s3";
-- reverse: create "repository_backup_rustic" table
DROP TABLE "repository_backup_rustic";
-- reverse: create "user" table
DROP TABLE "user";
-- reverse: create "repository" table
DROP TABLE "repository";
