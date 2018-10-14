//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
)

var (
	// Limit query parameter
	Limit uint = 10
	// Offset query parameter
	Offset uint
)

func dump(i interface{}) {
	fmt.Printf("%s", color.BlueString(spew.Sdump(i)))
}

func logUsage(u string) {
	fmt.Printf(color.YellowString("Usage:  %s\n"), u)
}

func logError(err error) {
	fmt.Printf("%s\n", color.RedString(err.Error()))
}

func logOK() {
	fmt.Printf("%s\n", color.GreenString("OK"))
}
