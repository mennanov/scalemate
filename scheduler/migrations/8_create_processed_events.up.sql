create table processed_events
(
    event_uuid bytea,
    retries    int       not null default 0,
    created_at timestamp not null default now(),
    updated_at timestamp,
    primary key (event_uuid)
);
