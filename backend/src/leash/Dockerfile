FROM golang:1.24-alpine

WORKDIR /app

RUN touch .env

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY ./src ./src

RUN go build -o /leash src/leash/leash.go

EXPOSE 8000

CMD [ "/leash", "launch" ]