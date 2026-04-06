// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package nats hold the implementation of the Publisher and PubSub
// interfaces for the NATS messaging system, the internal messaging
// broker of the Magistrala IoT platform. Due to the practical requirements
// implementation Publisher is created alongside PubSub. The reason for
// this is that Subscriber implementation of NATS brings the burden of
// additional struct fields which are not used by Publisher. Subscriber
// is not implemented separately because PubSub can be used where Subscriber is needed.
package nats
