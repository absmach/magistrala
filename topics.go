// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mainflux

// File topics.go contains all NATS subjects that are shared between services.

// OutputSenML represents subject SenML messages will be published to.
// Messages published to this topic are Protobuf serialized Messages.
const OutputSenML = "out.senml"

// InputChannels represents subject all messages will be published to.
// Messages received on this topic are Protobuf serialized RawMessages.
const InputChannels = "channel.>"
