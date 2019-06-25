create table container_labels
(
  container_id int REFERENCES containers (id) on update cascade on delete cascade,
  label        text,
  primary key (container_id, label)
);
