FROM golang:1.26.7-alpine@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468

WORKDIR /app

RUN touch .env

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY ./src ./src

RUN go build -o /leash src/leash/leash.go

EXPOSE 8000

CMD [ "/leash", "launch" ]
