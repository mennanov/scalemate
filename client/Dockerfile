FROM golang:alpine AS build-env

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh make protobuf

RUN go get -u github.com/golang/dep/cmd/dep

# Set up SSH keys for accessing github private repositories (github.com/mennanov/scalemate-shared).
ARG ID_RSA
ARG ID_RSA_PUB

RUN mkdir -p /root/.ssh && chmod 0700 /root/.ssh
RUN printf "%b" "${ID_RSA}" > /root/.ssh/id_rsa
RUN chmod 600 /root/.ssh/id_rsa
RUN printf "%b" "${ID_RSA_PUB}" > /root/.ssh/id_rsa.pub
RUN eval "$(ssh-agent)" && ssh-add /root/.ssh/id_rsa
RUN ssh-keyscan -t rsa github.com >> /root/.ssh/known_hosts

WORKDIR /go/src/github.com/mennanov/scalemate-client
COPY . /go/src/github.com/mennanov/scalemate-client

RUN dep ensure
RUN make build

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build-env /go/src/github.com/mennanov/scalemate-client/build/scalemate /bin/scalemate
ENTRYPOINT ["scalemate"]
