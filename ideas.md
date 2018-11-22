## Client

Important idea: scalemate user should be able to run any command remotely 
while simulating local execution. For example:

```text
> sm run -d postgres
# Remote container ports are p2p mapped to localhost.
> psql -h localhost -p 5432
```

`~/.sm.yml:`
```yaml

confs:
    - name: postgres
    image: postgres:10
    cpu_limit: 1
    disk_limit: 2GB
    memory_limit: 512MB
    labels: # Nodes labels to schedule this on.
      - scalemate-beta # free servers
      - tesla-v100
      - intel-i7
    ports:
      - 5432:5432
```

A separate Bash wrapper around scalemate api client:

Aliases are configured in conf files like ~/.sm.yml

For e.g. in order to run
```bash
docker run \
  -e USER="$(id -u)" \
  -u="$(id -u)" \
  -v /src/workspace:/src/workspace \
  -v /tmp/build_output:/tmp/build_output \
  -w /src/workspace \
  l.gcr.io/google/bazel:0.17.1 \
  --output_user_root=/tmp/build_output \
  build //absl/...
```

It needs to be configured as:
```yaml

confs:
    - name: bazel
    image: l.gcr.io/google/bazel:0.17.1
    
    env:
      - USER="$(id -u)"
    user: "$(id -u)" # bash substitution for the current client shell
    volumes:
      - /src/workspace:/src/workspace
      - /tmp/build_output:/tmp/build_output
    work_dir: /src/workspace
    args: "--output_user_root=/tmp/build_output"
    cpu_limit: 4
    cpu_class: advanced
    disk_limit: 2GB
    memory_limit: 2GB
```

Then in terminal:

```text
> sm bazel build //absl/...
Getting current dir context… 25 MB
Scheduled on the node mennanov/server1
Establishing p2p connection…
Sending context 25 MB…
[Bazel runs below]
INFO: Analysed 72 targets (0 packages loaded).
INFO: Found 72 targets...
INFO: Elapsed time: 0.425s, Critical Path: 0.00s
[Bazel finished]
Downloading output 134 MB…
```


## Server

Agent will be able to be configured in such a way that it will allow certain users to run
tasks for free:

`~/.sm-server.yml`
```yaml

allow_free_usage: # usernames that are allowed to use this server free of charge
    - mennanov
    - johnny

```