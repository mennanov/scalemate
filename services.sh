#!/usr/bin/env bash
# This script is an alternative to a docker-compose.yml which would define all the services needed to run all the
# microservices and also tests.
# Those services need to run in the host network in order for Bazel to reach them, but `network_mode: host` in
# docker-compose.yml does not work on OS X: https://github.com/docker/for-mac/issues/1031
# Env variables with connection settings are set in .bazelrc

# RabbitMQ
if [[ `docker ps -aqf 'name=rabbit-shared'` == "" ]]; then
    docker run -d --rm --name rabbit-shared -p 15672:5672 rabbitmq:3-management
else
    echo "rabbit-shared is already running"
fi

# Postgres (accounts)
if [[ `docker ps -aqf 'name=postgres-accounts'` == "" ]]; then
    docker run -d --rm --name postgres-accounts -p 25432:5432 --tmpfs /var/lib/postgresql/data postgres:11
else
    echo "postgres-accounts is already running"
fi

# Postgres (scheduler)
if [[ `docker ps -aqf 'name=postgres-scheduler'` == "" ]]; then
    docker run -d --rm --name postgres-scheduler -p 35432:5432 --tmpfs /var/lib/postgresql/data postgres:11
else
    echo "postgres-scheduler is already running"
fi
