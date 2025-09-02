create table volume
(
    id            TYPES_INT_PRIMARY_KEY,
    created_at    TYPES_DATETIME not null default current_timestamp,
    deleted_at    TYPES_DATETIME,
    finalizers    text           not null default '{}',

    repository_id bigint         not null references repository (id) on delete restrict,

    uuid          text           not null unique,
    name          text           not null,

    fs_size       bigint         not null,
    fs_type       text           not null,

    lock_id       text,
    lock_time     bigint,

    unique (repository_id, uuid),
    unique (repository_id, name)
);
