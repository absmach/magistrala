// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/absmach/magistrala/internal/clients/grpc"
	mflog "github.com/absmach/magistrala/logger"
)

func Close(log mflog.Logger, clientHandler grpc.ClientHandler) {
	if err := clientHandler.Close(); err != nil {
		log.Warn(err.Error())
	}
}
