FROM scalemate-accounts:base AS build-env
COPY ./ /go/src/github.com/mennanov/scalemate/accounts/
WORKDIR /go/src/github.com/mennanov/scalemate/accounts/
RUN dep ensure
RUN make build

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build-env /go/src/github.com/mennanov/scalemate/accounts/build/accounts /bin/accounts
ENTRYPOINT ["/bin/accounts"]
