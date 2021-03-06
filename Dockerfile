FROM golang:1.16.2

WORKDIR /source
ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-w " -o sync main.go

# Second stage - minimal image
FROM alpine

RUN apk update && apk add --no-cache git
COPY --from=0 /source/sync /app/sync

ENTRYPOINT ["/app/sync"]