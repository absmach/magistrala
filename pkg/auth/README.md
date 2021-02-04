# Auth client

Auth client package is used to identify a thing and authorize a thing's access to a channel.

To identify a thing, you need a valid **thing key**. You retrieve thing's identity in the form of a **thing ID**. The latter is used in CRUD operations on things and their connections.

To authorize a thing's access to a channel, you need a valid **thing ID** and a valid **channel ID**. If a thing is not connected to a channel, the auth client responds with an error. Otherwise, a *nil* value is returned, signaling the successful authorization.
