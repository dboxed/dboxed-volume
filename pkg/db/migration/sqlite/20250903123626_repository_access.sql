-- +goose Up
-- create "repository_access" table
CREATE TABLE `repository_access` (
  `repository_id` bigint NOT NULL,
  `user_id` text NOT NULL,
  PRIMARY KEY (`repository_id`, `user_id`),
  CONSTRAINT `0` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT `1` FOREIGN KEY (`repository_id`) REFERENCES `repository` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);

-- +goose Down
-- reverse: create "repository_access" table
DROP TABLE `repository_access`;
