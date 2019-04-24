create table nodes
(
  id                       serial
    constraint node_pk primary key,
  username                 text      not null,
  name                     text      not null,
  status                   smallint  not null,
  cpu_capacity             smallint  not null,
  cpu_available            smallint  not null,
  cpu_class                smallint  not null,
  memory_capacity          integer   not null,
  memory_available         integer   not null,
  gpu_capacity             smallint  not null,
  gpu_available            smallint  not null,
  gpu_class                smallint  not null,
  disk_capacity            integer   not null,
  disk_available           integer   not null,
  disk_class               smallint  not null,
  network_ingress_capacity smallint  not null,
  network_egress_capacity  smallint  not null,
  containers_finished      bigint    not null,
  containers_failed        bigint    not null,
  connected_at             timestamp,
  disconnected_at          timestamp,
  last_scheduled_at        timestamp,
  created_at               timestamp not null default now(),
  updated_at               timestamp,
  ipv4_address             inet,
  unique (username, name)
);

CREATE TRIGGER nodes_set_updated_at
  BEFORE UPDATE
  ON nodes
  FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
