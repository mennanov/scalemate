# scalemate-shared
Scalemate.io shared entities (e.g. .proto files and other tools)

### Running tests

1. `dep ensure`
2. `make test`

### Running a Drone instance locally

`drone exec`

### Running linters

1. `go get -u gopkg.in/alecthomas/gometalinter.v2 && gometalinter.v2 --install`
2. `make lint`
