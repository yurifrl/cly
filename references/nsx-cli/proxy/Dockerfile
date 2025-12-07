# Build stage
FROM golang:1.24 AS build
WORKDIR /app
ARG GHA_PAT

RUN mkdir -p /app/bin

ENV GO111MODULE=on
ENV GOPRIVATE "github.com/NSXBet/*"
ENV GHA_PAT=$GHA_PAT
RUN git config --global url."https://$GHA_PAT@github.com/".insteadOf "https://github.com/"

COPY . .
RUN go mod download all

ENV ROOT_PATH /app
RUN go build -o /app/bin/cli ./proxy/main.go
RUN chmod +x /app/bin/cli

# Runtime stage
FROM gcr.io/distroless/base-nossl-debian12
WORKDIR /app
COPY --from=build /app/bin/cli /app/cli
ENTRYPOINT [ "/app/cli" ]
