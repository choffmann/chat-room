#############################################
# Preparer go
#############################################
FROM golang:1.25-alpine AS preparer_go

WORKDIR /app/build

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .

#############################################
# Builder go
#############################################
FROM preparer_go AS builder

ARG APP_VERSION="v0.0.0"
ARG APP_GIT_COMMIT="unknown"
ARG APP_GIT_BRANCH="main"
ARG APP_GIT_REPOSITORY="https://github.com/choffmann/chat-room"
ARG APP_BUILD_TIME="unknown"

RUN go build -o "bin/chat-room" \
    -ldflags=" \
      -s -w \
      -X main.version=${APP_VERSION} \
      -X main.gitCommit=${APP_GIT_COMMIT} \
      -X main.gitBranch=${APP_GIT_BRANCH} \
      -X main.gitRepository=${APP_GIT_REPOSITORY} \
      -X main.buildTime=${APP_BUILD_TIME} \
    "

#############################################
# Runner go
#############################################
FROM alpine:3.22 AS runner

EXPOSE 8080

RUN adduser -D gorunner

USER gorunner
WORKDIR /app

COPY --chown=gorunner:gorunner --from=builder /app/build/bin/chat-room /app/chat-room

ENTRYPOINT [ "/app/chat-room" ]
