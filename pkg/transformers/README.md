# Message Transformers

Transformers services consume events published by adapters and transform them to any other message format.
They be imported as a standalone package and used for message transformation on the consumer side.
Mainflux [SenML transformer](transformer) is an example of Transformer service for SenML messages.
Mainflux [writers](writers) are using a standalone SenML transformer to preprocess messages before storing them.

[transformers]: https://github.com/mainflux/mainflux/tree/master/transformers/senml
[writers]: https://github.com/mainflux/mainflux/tree/master/writers
