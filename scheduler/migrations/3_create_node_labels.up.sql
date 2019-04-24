create table node_labels
(
  node_id int REFERENCES nodes (id) on update cascade on delete cascade,
  label   text,
  primary key (node_id, label)
);

