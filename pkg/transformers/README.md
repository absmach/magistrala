# Message Transformers

A transformer service consumes events published by Magistrala adapters (such as MQTT and HTTP adapters) and transforms them to an arbitrary message format. A transformer can be imported as a standalone package and used for message transformation on the consumer side.

Magistrala [SenML transformer](transformer) is an example of Transformer service for SenML messages.

Magistrala [writers](writers) are using a standalone SenML transformer to preprocess messages before storing them.

[transformers]: https://github.com/absmach/magistrala/tree/master/transformers/senml
[writers]: https://github.com/absmach/magistrala/tree/master/writers
