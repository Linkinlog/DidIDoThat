FROM golang:1.23.1 as builder

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN make build

FROM scratch

COPY --from=builder /app/ass /ass

CMD ["/ass"]
