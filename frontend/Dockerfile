FROM node:current-alpine3.18 AS base
WORKDIR /app
COPY package.json ./
COPY package-lock.json ./
RUN npm install


COPY . .

RUN npm run build

EXPOSE 3000

CMD ["node", "build"]