create table containers
(
  id                  bigserial
    constraint containers_pk
      primary key,
  node_id             int
    constraint containers_nodes_id_fk
      references nodes (id)
      on update cascade on delete restrict,
  username            text      not null,
  status              smallint  not null,
  status_message      text      not null,
  image               text      not null,
  cpu_class_min       smallint  not null,
  cpu_class_max       smallint  not null,
  gpu_class_min       smallint  not null,
  gpu_class_max       smallint  not null,
  disk_class_min      smallint  not null,
  disk_class_max      smallint  not null,
  network_ingress_min smallint  not null,
  network_egress_min  smallint  not null,

  created_at          timestamp not null default now(),
  updated_at          timestamp,

  agent_auth_token    bytea     not null
);

CREATE TRIGGER containers_set_updated_at
  BEFORE UPDATE
  ON containers
  FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
