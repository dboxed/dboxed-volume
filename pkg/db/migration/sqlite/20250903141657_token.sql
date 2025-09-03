-- +goose Up
-- create "token" table
CREATE TABLE `token` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NOT NULL DEFAULT (current_timestamp),
  `token` text NOT NULL,
  `name` text NOT NULL,
  `user_id` text NOT NULL,
  CONSTRAINT `0` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "token_token" to table: "token"
CREATE UNIQUE INDEX `token_token` ON `token` (`token`);

-- +goose Down
-- reverse: create index "token_token" to table: "token"
DROP INDEX `token_token`;
-- reverse: create "token" table
DROP TABLE `token`;
