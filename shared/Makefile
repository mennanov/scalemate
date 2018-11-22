# vi: ft=make

GOPATH:=$(shell go env GOPATH)

.PHONY: test proto lint mocks

proto:
	protoc -I proto/accounts -I ${GOPATH}/src accounts.proto --go_out=plugins=grpc:proto/accounts --govalidators_out=proto/accounts
	protoc -I proto/scheduler -I ${GOPATH}/src scheduler.proto --go_out=plugins=grpc:proto/scheduler --govalidators_out=proto/scheduler
	protoc -I proto/events -I ${GOPATH}/src events.proto --go_out=plugins=grpc:proto/events

test:
	go test -p 1 -coverprofile=coverage.coverage -v ./...

mocks:
	mockgen github.com/mennanov/scalemate-shared/proto/accounts AccountsClient > proto/accounts/mocks/accounts.mock.go
	mockgen github.com/mennanov/scalemate-shared/proto/scheduler SchedulerClient > proto/scheduler/mocks/scheduler.mock.go

lint:
	gometalinter.v2 --vendor --deadline=2m --disable-all --enable=vet --enable=gofmt --enable=golint --enable=gotype \
	--enable=megacheck --exclude="error should be the last type when returning multiple items \(golint\)" ./...

