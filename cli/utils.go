//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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
	Offset uint
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
	fmt.Printf("\n%s\n\n", color.RedString(err.Error()))
}

func logOK() {
	fmt.Printf("\n%s\n\n", color.BlueString("ok"))
}

func logCreated(e string) {
	fmt.Printf(color.BlueString("\ncreated: %s\n\n"), e)
}
