create table processed_events
(
  handler_name text,
  uuid         bytea,
  primary key (handler_name, uuid)
);

