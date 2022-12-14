FROM golang:alpine AS build

RUN apk add --no-cache git

WORKDIR /go/src/app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o ./collab-api .

FROM alpine

WORKDIR /app

RUN apk add ca-certificates

COPY --from=build /go/src/app/collab-api /app/collab-api

EXPOSE 8080

CMD ["/app/collab-api"]