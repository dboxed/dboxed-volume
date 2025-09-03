-- +goose Up
-- create "repository_access" table
CREATE TABLE "repository_access" (
  "repository_id" bigint NOT NULL,
  "user_id" text NOT NULL,
  PRIMARY KEY ("repository_id", "user_id"),
  CONSTRAINT "repository_access_repository_id_fkey" FOREIGN KEY ("repository_id") REFERENCES "repository" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "repository_access_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

-- +goose Down
-- reverse: create "repository_access" table
DROP TABLE "repository_access";
