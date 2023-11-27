// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
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
)

func logJSON(iList ...interface{}) {
	for _, i := range iList {
		m, err := json.Marshal(i)
		if err != nil {
			logError(err)
			return
		}

		pj, err := prettyjson.Format(m)
		if err != nil {
			logError(err)
			return
		}

		fmt.Fprintf(os.Stdout, "\n%s\n\n", string(pj))
	}
}

func logUsage(u string) {
	fmt.Fprintf(os.Stdout, color.YellowString("\nusage: %s\n\n"), u)
}

func logError(err error) {
	boldRed := color.New(color.FgRed, color.Bold)
	boldRed.Fprintf(os.Stderr, "\nerror: ")

	fmt.Fprintf(os.Stderr, "%s\n\n", color.RedString(err.Error()))
}

func logOK() {
	fmt.Fprintf(os.Stdout, "\n%s\n\n", color.BlueString("ok"))
}

func logCreated(e string) {
	if RawOutput {
		fmt.Fprintln(os.Stdout, e)
	} else {
		fmt.Fprintf(os.Stdout, color.BlueString("\ncreated: %s\n\n"), e)
	}
}

func logRevokedTime(t time.Time) {
	if RawOutput {
		fmt.Fprintln(os.Stdout, t)
	} else {
		fmt.Fprintf(os.Stdout, color.BlueString("\nrevoked: %v\n\n"), t)
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
