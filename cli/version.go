//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get version of Mainflux Things Service",
		Long:  `Mainflux server health checkt.`,
		Run: func(cmd *cobra.Command, args []string) {
			Version()
		},
	}
}

// Version - server health check
func Version() {
	url := fmt.Sprintf("%s/version", serverAddr)
	FormatResLog(httpClient.Get(url))
}
