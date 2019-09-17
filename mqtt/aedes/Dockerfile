# Copyright (c) Mainflux
# SPDX-License-Identifier: Apache-2.0

FROM node:10.15.1-alpine as builder

## Install build toolchain, install node deps and compile native add-ons
RUN apk add --no-cache python make g++
COPY mqtt/aedes/package.json .
RUN npm rebuild && npm install --only=production

FROM node:10.15.1-alpine
## Copy built node modules and binaries without including the toolchain
COPY --from=builder node_modules/ ./node_modules
COPY *.proto mqtt/* ./

EXPOSE 1883 8880

CMD ["node", "mqtt.js"]