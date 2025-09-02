create table repository
(
    id         TYPES_INT_PRIMARY_KEY,
    created_at TYPES_DATETIME not null default current_timestamp,
    deleted_at TYPES_DATETIME,
    finalizers text           not null default '{}',

    name       text           not null,
    uuid       text           not null unique,

    unique (name)
);

create table repository_storage_s3
(
    id                bigint primary key references repository (id) on delete cascade,

    endpoint          text not null,
    region            text,
    bucket            text not null,
    access_key_id     text not null,
    secret_access_key text not null,
    prefix            text not null
);

create table repository_backup_rustic
(
    id       bigint primary key references repository (id) on delete cascade,

    password text not null
);
