create table node_pricing
(
    id           bigserial
        constraint nodes_pricing_pk primary key,
    node_id      bigint    not null
        constraint containers_nodes_id_fk
            references nodes (id)
            on update cascade on delete restrict,
    cpu_price    bigint    not null,
    memory_price bigint    not null,
    gpu_price    bigint    not null,
    disk_price   bigint    not null,
    created_at   timestamp not null default now()
);

CREATE INDEX node_pricing_idx ON node_pricing (node_id, created_at DESC);
