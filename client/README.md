# Scalemate.io CLI client

### Building a Docker image

`docker build -t scalemate-client --build-arg ID_RSA="$(cat path/to/id_rsa)" --build-arg ID_RSA_PUB="$(cat path/to/id_rsa.pub"  .`

### Running tests

1. `dep ensure`
2. `make test`

### Running a Drone instance locally

`drone exec --secret ID_RSA=$ID_RSA --secret ID_RSA_PUB=$ID_RSA_PUB`

where `$ID_RSA` and `$ID_RSA_PUB` are the corresponding deployment keys
for the [github.com/mennanov/scalemate-shared] repository.

### Running linters

1. `go get -u gopkg.in/alecthomas/gometalinter.v2 && gometalinter.v2 --install`
2. `make lint`
