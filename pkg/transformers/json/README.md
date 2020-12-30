# JSON Message Transformer

JSON Transformer provides Message Transformer for JSON messages.
To transform Mainflux Message successfully, the payload must be a JSON object.

For the messages that contain _JSON array as the root element_, JSON Transformer does normalization of the data: it creates a separate JSON message for each JSON object in the root. In order to be processed and stored properly, JSON messages need to contain message format information. For the sake of the simpler storing of the messages, nested JSON objects are flatten to a single JSON object, using composite keys with the default separator `/`. This implies that the separator character (`/`) _is not allowed in the JSON object key_. For example, the following JSON object:
```json
{
    "name": "name",
    "id":8659456789564231564,
    "in": 3.145,
    "alarm": true,
    "ts": 1571259850000,
    "d": {
        "tmp": 2.564,
        "hmd": 87,
        "loc": {
            "x": 1,
            "y": 2
        }
    }
}
```

will be transformed to:

```json

{
    "name": "name",
    "id":8659456789564231564,
    "in": 3.145,
    "alarm": true,
    "ts": 1571259850000,
    "d/tmp": 2.564,
    "d/hmd": 87,
    "d/loc/x": 1,
    "d/loc/y": 2
}
```

The message format is stored in *the subtopic*. It's the last part of the subtopic. In the example:

```
http://localhost:8185/channels/<channelID>/messages/home/temperature/myFormat
```

the message format is `myFormat`. It can be any valid subtopic name, JSON transformer is format-agnostic. The format is used by the JSON message consumers so that they can process the message properly. If the format is not present (i.e. message subtopic is empty), JSON Transformer will report an error. Since the Transformer is agnostic to the format, having format in the subtopic does not prevent the publisher to send the content of different formats to the same subtopic. It's up to the consumer to handle this kind of issue. Message writers, for example, will store the message(s) in the table/collection/measurement (depending on the underlying database) with the name of the format (which in the example is `myFormat`). Mainflux writers will try to save any format received (whether it will be successful depends on the writer implementation and the underlying database), but it's recommended that the publisher takes care not to send different formats to the same subtopic.

Having a message format in the subtopic means that the subscriber has an option to subscribe to only one message format. This is a nice feature because message subscribers know what's the expected format of the message so that they can process it. If the message format is not important, wildcard subtopic can always be used to subscribe to any message format:

```
http://localhost:8185/channels/<channelID>/messages/home/temperature/*
```
