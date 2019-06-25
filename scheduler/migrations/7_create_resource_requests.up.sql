create table resource_requests
(
    id             bigserial
        constraint resource_requests_pk
            primary key,
    container_id   int       not null
        constraint resource_requests_containers_id_fk
            references containers (id)
            on update cascade on delete restrict,
    cpu            smallint  not null,
    memory         int   not null,
    disk           int   not null,
    gpu            smallint  not null,
    status         smallint  not null,
    status_message text,

    created_at     timestamp not null default now(),
    updated_at     timestamp
);

CREATE TRIGGER resource_requests_set_updated_at
    BEFORE UPDATE
    ON resource_requests
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
