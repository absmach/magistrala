# Mainflux Changelog

## Generation
Mainflux release notes for the latest release can be obtained via:
```
make changelog
```

Otherwise, whole log in a similar format can be observed via:
```
git log --pretty=oneline --abbrev-commit
```

## 0.7.0 - 08. DEC 2018.
### Features

- MF-486 - Add provisioning command to CLI (#487)
- Fix lora-adapter event store handlers (#492)
- NOISSUE - Add LoRa route map validation and fix LoRa messages URL (#491)
- MF-475 - Replace increment ID with UUID (#490)
- MF-166 - Add lora-adapter service (#481)
- NOISSUE - Add Makefile target to clean old imgs (#485)
- MF-473 -  Add metadata field to channel (#476)
- Make CoAP ping period configurable (#469)
- Add nginx ingress config to k8s services (#472)
- Add CoAP section in getting-started (#468)
- NOISSUE - Move CLI documentation from getting started guide to separate page (#470)
- NOISSUE - Update Getting Started doc with CLI usage (#465)
- Update CoAP docs with URL example (#463)
- MF-447 - Add event sourcing to things service (#460)
- Add TLS support to CoAP adapter and all readers (#459)
- MF-417 - Implement SDK tests (#438)
- MF-454 - Use message Time field as a time for InfluxDB points (#455)
- NOISSUE - Add .dockerignore to project root (#457)
- Update docker-compose so that every service has debug log level (#453)
- NOISSUE - Add TLS flag for Mainflux services (#452)
- MF-448 - Option for Postgres SSL Mode (#449)
- MF-443 Update project dependencies (#444)
- MF-426 - Add optional MF_CA_CERTS env variable to allow GRPC client to use TLS certs (#430)
- Expose the InfluxDB and Cassandra ports to host (#441)
- MF-374 - Bring back CoAP adapter (#413)

### Bugfixes
- gRPC Load Balancing between http-adapter and things (#387)
- MF-407 - Values of zero are being omitted  (#434)

### Summary
https://github.com/mainflux/mainflux/milestone/8?closed=1


## 0.6.0 - 26. OCT 2018.
### Features 

- Added Go SDK (#357)
- Updated NATS version (#412)
- Added debbug level to MFX logger (#379)
- Added Documentation for readers (#389)
- Added Redis cache to improve performance (#382)


## 0.5.1 - 05. SEP 2018.
### Features
- Improve performance by adding Redis cache (#382)

### Bugfixes
- Mixed up name and type of the things (#375)
- Fix MQTT topic (#380)


## 0.5.0 - 28. AUG 2018
### Features
- InfluxDB Reader (#311)
- Cassandra Reader (#313)
- MongoDB Reader (#344)
- MQTT Persistance via Redis (#328)
- CLI integrated into monorepo (#216) 
- Normalizer logging (#333)
- WS swagger doc (#337)
- Payload renamed to Metadata (#343)
- Protobuf files added (#363)
- SPDX headers added (#325)

### Bugfixes
- Docker network for InfluxDB (#346)
- Vendor correct gRPC version (#340)

### Summary
https://github.com/mainflux/mainflux/milestone/6?closed=1


## 0.4.0 - 01. JUN 2018.
* Integrated MQTT adapter (#165 )
* Support for storing messages in MongoDB (#237) 
* Support for storing messages in InfluxDB (#236)
* Use UUID PKs with auto-incremented values (#269 )
* Replaced JWT with plain string tokens in things service (#268 ) 
* Emit non-SenML messages (#239 )
* Support for Grafana (#296)
* Added WS Load test (#299 )


## 0.3.0 - 14. MAY 2018.
- CoAP API for message exchange (#186)
- Split `manager` service into `clients` and `users` (#266)
- Replaced ORM with raw SQL (#265)
- Setup Kubernetes (#226, #273)
- Fix docker compose (#274)
- Integrated `dashflux` into monorepo (#258)
- Integrated (*non-compatible*) `mqtt` into monorepo (#260)


## 0.2.3 - 24. APR 2018.
- Fix examples in the documentation (#243)
- Add service name in info response (#241)
- Improve code coverage in WS adapter (#242)


## 0.2.2 - 23. APR 2018.
- Setup load testing scenarios (#225)


## 0.2.1 - 22. APR 2018.
- Fixed `Content-Type` header checking (#238)

## 0.2.0 - 18. APR 2018
- Protobuf message serialization (#192)
- Websocket API for exchanging messages (#188)
- Channel & client retrieval paging (#227) 
- Service instrumentation (#213)
- `go-kit` based JSON logger (#212)
- Project documentation (#218, #220)
- API tests (#211, #224)


## 0.1.2 - 18. MAR 2018.
### Bug fixes

- Fixed go lint warnings (#189)
- Compose failing startup (#185)
- Added missing service startup messages (#190)