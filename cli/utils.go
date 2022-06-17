// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	prettyjson "github.com/hokaccha/go-prettyjson"
)

var (
	// Limit query parameter
	Limit uint = 10
	// Offset query parameter
	Offset uint = 0
	// Name query parameter
	Name string = ""
	// Email query parameter
	Email string = ""
	// Metadata query parameter
	Metadata string = ""
	// ConfigPath config path parameter
	ConfigPath string = ""
	// RawOutput raw output mode
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

		fmt.Printf("\n%s\n\n", string(pj))
	}
}

func logUsage(u string) {
	fmt.Printf(color.YellowString("\nusage: %s\n\n"), u)
}

func logError(err error) {
	boldRed := color.New(color.FgRed, color.Bold)
	boldRed.Print("\nerror: ")

	fmt.Printf("%s\n\n", color.RedString(err.Error()))
}

func logOK() {
	fmt.Printf("\n%s\n\n", color.BlueString("ok"))
}

func logCreated(e string) {
	if RawOutput {
		fmt.Println(e)
	} else {
		fmt.Printf(color.BlueString("\ncreated: %s\n\n"), e)
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
