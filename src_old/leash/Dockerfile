
FROM node:lts-alpine3.17 AS build

COPY ./src/leash/frontend/web/package.json ./
COPY ./src/leash/frontend/web/package-lock.json ./
RUN npm install

COPY ./src/leash/frontend/web/assets ./assets
COPY ./src/leash/frontend/web/src ./src
COPY ./src/leash/frontend/web/static ./static
COPY ./src/leash/frontend/web/tsconfig.json ./
COPY ./src/leash/frontend/web/svelte.config.js ./
COPY ./src/leash/frontend/web/vite.config.ts ./

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