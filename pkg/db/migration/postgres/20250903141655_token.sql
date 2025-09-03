-- +goose Up
-- create "token" table
CREATE TABLE "token" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "token" text NOT NULL,
  "name" text NOT NULL,
  "user_id" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "token_token_key" UNIQUE ("token"),
  CONSTRAINT "token_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- +goose Down
-- reverse: create "token" table
DROP TABLE "token";
