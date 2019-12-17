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


## 0.10.0 - 17. DEC 2019.
### Features
- MF-932 - User API keys (#941)
- NOISSUE - Use opcua server timestamp in opcua-adapter messages (#980)
- Simplify CI script (#979)
- NOISSUE - Add opcua-adapter conn route-map, use ServerURI and NodeID (#975)
- Move docs to a separate repo (#976)
- NOISSUE - Support multiple types values in opcua-adapter (#973)
- Migrate from dep to go modules (#971)
- NOISSUE - Add Node IdentifierType config in opcua-adapter (#967)
- NOISSUE - Remove messages limit in influxdb-reader (#968)
- MF-898 - Add bulk connect to CLI and SDK (#956)
- MF-538 - Improve logging and API errors (#866)
- NOISSUE - Remove Elm UI (#953)
- MF-898 - Add bulk connections endpoint (#948)
- MF-898 - Change thing's service to use bulk connect (#946)
- MF-898 - Add transactions to postgres connect (#940)
- Add missing user service tests (#945)
- Remove Normalizer service from compose (#937)
- MF-919 - Mainflux message updates (#924)
- NOISSUE - Remove ARM multi-arch images (#929)
- MF-906 - Change single creation endpoints to use bulk service calls (#927)
- MF-922 - Add UpdateUser endpoint (#923)
- MF-780 - Use Normalizer as a lib (#915)
- NOISSUE - Switch to grpcbox for VerneMQ (#914)
- Change channels to chs (#918)
- MF-484 - Add bulk provisioning for things and channels (#889)
- MF-899 - Update README and official docs (#910)
- NOISSUE - Fix Redis envars (#903)
- Add disconnect on gen_server terminate() (#913)
- MF-890 - Add OPC-UA docs  (#904)
- NOISSUE - Update Protobuf version (#902)
- MF-886 - Add OPC-UA adapter (#878)
- MF-532 - Password reset (#873)
- MF-785 - Change CanAccess to CanAccessByKey (#894)
- NOISSUE - Add MQTT UserName check on register and InstanceId in Redis (#884)
- Add MQTT troubleshooting section (#882)
- MF-875 - Add tracing to official documentation (#877)
- MF-788 - Remove date and minimize copyright comments (#876)
- MF-787 - Add tags to user, thing, and channel spans (#869)
- Update docker-compose version for addons (#874)
- MF-859 - Channels metadata search (#867)
- MF-858 Users metadata (#861)
- NOISSUE - Simplify MQTT benchmarking tool (#852)
- NOISSUE - Upgrade Go version to 1.13 in container images (#868)
- MF-820 - Fetch messages for a particular device (#843)
- Update gorilla websocket version (#865)
- NOISSUE - Update aedes version and fix Dockerfile (#863)
- NOISSUE - Search by metadata (#849)
- MF-846 - Install python in docker build for aedes mqtt image (#860)
- NOISSUE - Clean NginX files, move .gitignores to dirs (#853)
- NOISSUE - Add docker-compose for MQTT cluster (#841)
- Add debug logs to the WS adapter (#848)
- NOISSUE - Add measuring time from pub to sub (#839)
- NOISSUE - update mqtt prov tool and some refactor (#831)
- NOISSUE - Use Thing ID to update certs data (#827)
- NOISSUE - Improve VerneMQ plugin code, add configurable gRPC pool size (#836)
- NOISSUE - Use gRPC for VerneMQ (#835)
- Switch secure of WS connection according to secure of http connection of UI (#829)
- NOISSUE - Use current hostname instead of localhost for a WebSocket connection in the UI (#826)
- NOISSUE - Improve MQTT benchmarking tools (#828)
- NOISSUE - update mqtt benchmark (#824)
- Add encryption key to env vars table (#823)
- NOISSUE - Add version endpoint to MQTT adapter (#816)
- MF-295 add mqtt benchmark tool (#817)
- update mqtts commands (#815)
- NOISSUE - Support encrypted bootstrap (#796)
- Add config to writers docs (#812)
- NOISSUE - Add VerneMQ support (#809)
- NOISSUE - Add content type as part of MQTT subscription topic (#810)


### Bugfixes
- Fix MQTT protobuf filename(#981)                                                                                                                                                                               
- MF-950 - Runtime error in normalizer - CBOR SenML (#974)
- NOISSUE - Fix opcua-adapter events warnings (#965)
- NOISSUE - Fix opcua-adapter events decode (#951)
- Fix subtopic handling in VerneMQ (#962)
- NOISSUE - Fix Update User (#959)
- NOISSUE - Fix make dockers (#957)
- Add dev_ back to make dockers_dev (#955)
- NOISSUE - Fix docs (#952)
- MF-916 - Fix Things and Channels counters (#947)
- MF-942 - Fix email template logic  (#944)
- NOISSUE - Fix HTTP header for Things and Channels creation (#939)
- NOISSUE - Fix docker ui image name (#938)
- NOISSUE - Fix lora-adapter (#936)
- NOISSUE - Fix lora creation events (#933)
- Fix doc for ENV vars in README (#920)
- Fix compilation (#911)
- Revert "NOISSUE - Make event sourcing optional (#907)" (#909)
- NOISSUE - Make event sourcing optional (#907)
- NOISSUE - Fix InfluxDB env vars (#908)
- Fix Elm version for ARM Docker images (#905)
- Fix Elm version in Dockerfile (#901)
- NOISSUE - fix security doc (#897)
- NOISSUE - Fix typo in docs and README (#891)
- Fix Nginx mTLS configuration (#885)
- Fix provision tool connect error handling (#879)
- Fix: Correct 404 and Content-Type Issues in MQTT Version Endpoint (#837)
- NOISSUE - Fix proto files in VerneMQ (#834)
- NOISSUE - Fix hackney HTTP request (#833)
- Add socket pool and fix pattern matching (#830)
- Fix typo (#814)

## 0.9.0 - 19. JUL 2019.
### Features
- Create and push docker manifest for new release from Makefile (#794)
- MF-399 - Add open tracing support (#782)
- MF-783 - Allow access checking by a thing ID (#784)
- NOISSUE - Add authorization HTTP API to things service (#772)
- Remove cli executable from repo (#776)
- NOISSUE - Use .env vars in docker-compose (#770)
- MF-663 - enable nginx port conf from docker env (#769)
- Update docs (#766)
- NOISSUE - Remove installing non-existent package in ci (#758)
-  NOISSUE - Add searchable Channels name  (#754)
- MF-466 - ARM docker deployment (#756)
- Add missing Websocket.js into docker ui image (#755)
- NOISSUE - Add searchable Things name (#750)
- NOISSUE - Add certificate fields to the Bootstrap service (#752)
- Update grpc and protobuf deps in mqtt adapter (#751)
- MF-742 - Things to support single user scenario (#749)
- MF-732 - Add Postgres reader (#740)
- MF-722 - Change UUID lib (#746)
- Add performance improvement to writer filtering (#744)
- NOISSUE - Update nginx version (#748)
- MF-574 - Add missing environment variables to Cassandra writer (#745)
- NOISSUE - Add compile test to CI (#743)
- MF-708 - Assign Writer(s) to a channel (#737)
- MF-732 - Add PostgreSQL writer (#733)
- NOISSUE - Add readers pagination in SDK (#736)
- Add UI websocket open/close and send/receive (#728)
- MF-707 - Allow custom Thing key (#726)
- MF-525 - Add pagination response to the readers (#729)
- NOISSUE - Rm Things type from lora-adapter (#727)
- skip deleting of persistent volumes by default (#723)
- MF-488 - Remove Thing type (app or device) (#718)
- Remove empty channels check (#720)
- MF-655 Proper usage of docker volumes (#657)
- NOISSUE - Improve UI styling (#719)
- MF-715 - Conflict on updating connection with a valid list of channels (#716)
- MF-711 - Create separate Redis instance for ES (#717)
- NOISSUE - Update event fields naming (#713)
- MF-698 - Add missing info and docs about sys event sourcing (#712)
- MF-549 - Change metadata format from JSON string to JSON object (#706)
- NOISSUE - Replace repeating code by card gen func (#697)
- Update Bootstrap service docker-compose.yml (#700)
- Remove Debug function (#699)
- MF-687 - Add event sourcing to Bootstrap service (#695)
- NOISSUE - Remove debugging message from response of handle error function (#696)
- Add event stream to MQTT adapter for conn status (#692)
- NOISSUE - Improve UI style (#691)
- Update docs structure (#686)
- Use images instead of carousel (#685)
- NOISSUE - Update docs (#683)
- MF-662 - Change menu style (#678)
- MF-651 - X509 Mutual TLS authentication (#676)
- Update Aedes version for MQTT adapter (#677)
- MF-661 - Bootstrap pagination in UI (#672)
- Update subtopics section in documentation (#670)
- Remove default base URL value (#671)

### Bugfixes
- NOISSUE - Fix Readers logs (#735)
- NOISSUE - Fix Docker for ARM (#760)
- NOISSUE - Fix count when search by name is performed (#767)
- NOISSUE - Typo fix (#777)
- NOISSUE - Fix Postgres logs in Things service (#734)
- Fix CI with fixed plugin versions (#747)
- fix building problems (#741)
- fix docker-compose env (#775)
- Fix MF_THINGS_AUTH_GRPC_PORT in addons' docker-compose files (#781)
- Fix MQTT raw message deserialization (#753)
- fix variant option for manifest annotate (#765)
- fix to makefile for OSX/Darwin (#724)
- Fix .dockerignore file by removing index.html (#725)
- Fix things and channels metadata create and edit & remove thing type (#721)
- Fix Bootstrap service event map keys (#705)
- Fix logging in publish event callback (#694)
- Fix InfluxDB time bug (#689)
- Fix users service to work in offline mode (#795)
- fix mainflux_id parameter in bootstrap swagger (#789)
- Fix offset calculation after deleting thing/channel, not to go to negative offset after deleting last thing/channel (#679)
- Use errors and null packets in authorized pub/sub (#773)
- NOISSUE - Fix CoAP adapter (#779)


### Summary
https://github.com/mainflux/mainflux/milestone/10?closed=1

## 0.8.0 - 20. MAR 2019.
### Features
- MF-571 - Add Env.elm to set custom base URL (#654)
- NOISSUE Added docs about docker-compose config overriding (#653)
- MF-539 - Improve Bootstrap Service documentation (#646)
- MF-596 - Add subtopic to RawMessage (#642)
- NOISSUE - Prevent infinite loop in lora-adapter if Redis init fail (#647)
- Corrected grammar and rephrased a few sentences to read nicely (#641)
- MF-571 - Elm UI (#632)
- MF-552 - Use event sourcing to keep Bootstrap service in sync with Things service (#603)
- MF-540 - Add pagination in API responses for Bootstrap service (#575)
- MF-600 - Handle custom LoRa Server application decoder (#608)
- update docker-compose (#590)
- Update generated code (#602)
- Add generated files check (#601)
- MF-597 - Removed legacy code as not needed anymore (#598)
- NOISSUE - Added normalizer service to run script (#594)
- Changed RawMessage (#587)
- NOISSUE - fix CLI log (#581)
- MF-519 - Refine Message (#567)
- NOISSUE - Add name field for Bootstrap Config (#564)
- Fix non-SenML message routing in normalizer (#573)
- NOISSUE - Update authors list (#569)
- Update lora.md (#568)
- NOISSUE- Improve LoRa doc (#562)
- MF-551 - Add metadata fields to Bootstrap Channels (#563)
- Fix MQTT adapter by setting subscription queue (#561)
- MF-558 - Add MQTT subtopics documentation (#559)
- Fix regexp for SUB (#557)
- Simplify MQTT topipc regexp (#555)
- MF-429 -Enabled MQTT subtopic's (#554)
- Add env var for number of concurrent messages (#545)
- NOISSUE - Update doc and fix empty key bug (#544)
- MF-370 - Simplify and refine CI (#541)
- NOISSUE - Add connection commands to CLI (#542)
- NOISSUE - Refine docs (#537)
- Update licnese year (#533)
- MF-513 - Add Bootstrapping service (#524)
- Add dedicated env vars for event sourcing (#536)
- NOISSUE - Fix docs (#535)
- Add lora doc to getting-started.md (#529)
- MF-483 - Enable channels and devices corresponding lists in backend (#520)
- Add missing components doc to architecture.md (#531)

### Bugfixes
- MF-639 Split Content-Type header field on semicolon and evaluate all substrings (#644)
- MF-656 - Change bootstrap service port to 8200 (#658)
- Replace crossOrigin with relative path and fix messaging bug (#645)
- MF-579 Things & Channels returns 404 when not found or ID is malformed, not 500 (#633)
- Fix run command in dev guide (#605)
- MF-583 - Correct cmd/mongodb-reader HTTPServer log Info (#584)
- Fix Dusan Maldenovic GitHub (#570)
- Fix CLI docs (#566)
- Fix pagination response for empty page (#547)
- Fix swagger and provisioning docs (#546)
- NOISSUE - Fix event sourcing client on LoRa adapter (#527)
- Fix MQTT adapter scaling issue (#526)
- NOISSUE - Fix subtopic regex and restrict empty subtopic parts (#659)
- Fix missing css in container ui (#638)
- NOISSUE - Fix lora-adapter Object decode (#610)
- NOISSUE - Fix users logs in main.go (#577)
- NOISSUE - Fix normalizer exposed port in docker-compose (#548)

### Summary
https://github.com/mainflux/mainflux/milestone/9?closed=1


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
