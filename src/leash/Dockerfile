
FROM node:lts-alpine3.17 AS build

COPY ./src/leash/web/package.json ./
COPY ./src/leash/web/package-lock.json ./
RUN npm install

COPY ./src/leash/web/assets ./assets
COPY ./src/leash/web/src ./src
COPY ./src/leash/web/static ./static
COPY ./src/leash/web/tsconfig.json ./
COPY ./src/leash/web/svelte.config.js ./
COPY ./src/leash/web/vite.config.ts ./

RUN npm run build


FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY ./src/ ./src

RUN go build -o /leash src/leash/leash.go

COPY --from=build ./build /web

EXPOSE 8000

CMD [ "/leash" ]