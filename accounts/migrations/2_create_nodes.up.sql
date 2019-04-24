create table nodes
(
  id          bigserial
    constraint nodes_pk
      primary key,
  username    text      not null,
  name        text      not null,
  fingerprint bytea     not null,
  created_at  timestamp not null default now(),
  updated_at  timestamp,
  UNIQUE (username, name)
);

CREATE TRIGGER nodes_set_updated_at
  BEFORE UPDATE
  ON nodes
  FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
