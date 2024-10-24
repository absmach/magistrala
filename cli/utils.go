// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"
)

var (
	// Limit query parameter.
	Limit uint64 = 10
	// Offset query parameter.
	Offset uint64 = 0
	// Name query parameter.
	Name string = ""
	// Identity query parameter.
	Identity string = ""
	// Metadata query parameter.
	Metadata string = ""
	// Status query parameter.
	Status string = ""
	// ConfigPath config path parameter.
	ConfigPath string = ""
	// State query parameter.
	State string = ""
	// Topic query parameter.
	Topic string = ""
	// Contact query parameter.
	Contact string = ""
	// RawOutput raw output mode.
	RawOutput bool = false
	// Username query parameter.
	Username string = ""
	// FirstName query parameter.
	FirstName string = ""
	// LastName query parameter.
	LastName string = ""
)

func logJSONCmd(cmd cobra.Command, iList ...interface{}) {
	for _, i := range iList {
		m, err := json.Marshal(i)
		if err != nil {
			logErrorCmd(cmd, err)
			return
		}

		pj, err := prettyjson.Format(m)
		if err != nil {
			logErrorCmd(cmd, err)
			return
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n\n", string(pj))
	}
}

func logUsageCmd(cmd cobra.Command, u string) {
	fmt.Fprintf(cmd.OutOrStdout(), color.YellowString("\nusage: %s\n\n"), u)
}

func logErrorCmd(cmd cobra.Command, err error) {
	boldRed := color.New(color.FgRed, color.Bold)
	boldRed.Fprintf(cmd.ErrOrStderr(), "\nerror: ")

	fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", color.RedString(err.Error()))
}

func logOKCmd(cmd cobra.Command) {
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n\n", color.BlueString("ok"))
}

func logCreatedCmd(cmd cobra.Command, e string) {
	if RawOutput {
		fmt.Fprintln(cmd.OutOrStdout(), e)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), color.BlueString("\ncreated: %s\n\n"), e)
	}
}

func logRevokedTimeCmd(cmd cobra.Command, t time.Time) {
	if RawOutput {
		fmt.Fprintln(cmd.OutOrStdout(), t)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), color.BlueString("\nrevoked: %v\n\n"), t)
	}
}

func convertMetadata(m string) (map[string]interface{}, error) {
	var metadata map[string]interface{}
	if m == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(Metadata), &metadata); err != nil {
		return nil, err
	}
	return nil, nil
}
