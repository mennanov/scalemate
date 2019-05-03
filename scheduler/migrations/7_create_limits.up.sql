create table limits
(
  id             bigserial
    constraint limits_pk
      primary key,
  container_id   int
    constraint limits_containers_id_fk
      references containers (id)
      on update cascade on delete restrict,
  cpu            smallint  not null,
  memory         integer   not null,
  disk           integer   not null,
  gpu            smallint  not null,
  status         smallint  not null,
  status_message text,

  created_at     timestamp not null default now(),
  updated_at     timestamp,
  confirmed_at   timestamp
);

CREATE TRIGGER limits_set_updated_at
  BEFORE UPDATE
  ON limits
  FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
