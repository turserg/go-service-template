FROM golang:1.26.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /out/server /server
COPY --from=builder /out/migrate /migrate
COPY migrations /migrations

CMD ["/server"]
