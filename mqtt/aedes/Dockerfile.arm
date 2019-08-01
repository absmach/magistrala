FROM arm32v7/node:10.16.0-stretch-slim

COPY qemu-arm-static /usr/bin

COPY *.proto mqtt/* ./

RUN npm rebuild && npm install

EXPOSE 1883 8880

CMD ["node", "mqtt.js"]

RUN rm /usr/bin/qemu-arm-static