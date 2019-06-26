create table container_scheduling_strategies
(
    container_id          int REFERENCES containers (id) on update cascade on delete cascade,
    position              smallint not null,
    strategy              int,
    difference_percentage smallint,
    primary key (container_id, position)
);
