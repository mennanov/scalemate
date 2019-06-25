create table users
(
    id            bigserial
        constraint users_pk
            primary key,
    username      text      not null,
    email         text      not null,
    banned        boolean   not null default false,
    password_hash bytea     not null,
    created_at    timestamp not null default now(),
    updated_at    timestamp,
    UNIQUE (username),
    UNIQUE (email)
);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
