FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY ./ ./

RUN go build -o /leash src/leash/leash.go

RUN apk add --update nodejs npm
RUN cd src/leash/web && npm install && npm run build
RUN adduser -D -g 'www' www
RUN mkdir /www && chown -R www:www /www
RUN cp -r src/leash/web/build/* /www
RUN chown -R www:www /www
RUN cd ../../../

EXPOSE 8080

RUN apk update && apk add nginx
COPY src/leash/nginx.conf /etc/nginx/nginx.conf
RUN chown -R www:www /var/lib/nginx

RUN echo "nginx" >> start.sh
RUN echo "/leash" >> start.sh
RUN chmod +x start.sh

CMD ./start.sh