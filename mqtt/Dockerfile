FROM node:boron-alpine
MAINTAINER Mainflux

COPY *.proto mqtt/* ./

RUN npm rebuild && npm install

EXPOSE 1883 8880

CMD ["node", "mqtt.js"]
