###
# Mainflux Dockerfile
###
# Set the base image to Node, onbuild variant: https://registry.hub.docker.com/_/node/

FROM node:boron-alpine
MAINTAINER Mainflux

COPY . .
RUN npm install

EXPOSE 1883
EXPOSE 8880

###
# Run main command with dockerize
###
CMD ["node", "mqtt.js"]
