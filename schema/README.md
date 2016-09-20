## JSON Modles
Mainflux uses 2 entities in the system to represent devices and their properties:
- device
- channel

**Device** represents device itself - model, type, serial number, location...

**Channel** represents an observable property of device that we measure - temperature, pressure, light, velocity...

### Creating JSON Schema From the Model Templates
We use `deviceTemplate.json` and `channelTemplate.json` to describe our entities. Then based on these files
we can create more decriptive documents - [JSON Schemas](http://json-schema.org/).

To do this we can use on-line tool [http://jsonschema.net/](http://jsonschema.net/) or `npm` package `json-schema-generator`:
```bash
sudo npm install -g json-schema-generator
json-schema-generator ./deviceTemplate.json -o deviceSchema.json
```

Schemas will be used to perform JSON schema validation during API calls, as described in [this article](http://www.litixsoft.de/english/mms-json-schema/)
