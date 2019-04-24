create table containers
(
  id               serial
    constraint containers_pk
      primary key,
  node_id          int
    constraint containers_nodes_id_fk
      references nodes (id)
      on update cascade on delete restrict,
  username         text      not null,
  status           smallint  not null,
  status_message   text,
  image            text      not null,
  cpu_class_min    smallint,
  cpu_class_max    smallint,
  gpu_class_min    smallint,
  gpu_class_max    smallint,
  disk_class_min   smallint,
  disk_class_max   smallint,

  created_at       timestamp not null default now(),
  updated_at       timestamp,

  agent_auth_token bytea
);

CREATE TRIGGER containers_set_updated_at
  BEFORE UPDATE
  ON containers
  FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
