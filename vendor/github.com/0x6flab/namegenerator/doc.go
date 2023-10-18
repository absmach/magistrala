/*
Package namegenerator provides a simple way to generate random names.

Example to generate general names:

	package main

	import (
		"fmt"

		"github.com/0x6flab/namegenerator"
	)

	func main() {
		generator := namegenerator.NewNameGenerator("")
		name := generator.Generate()
		fmt.Println(name)
	}
*/
package namegenerator
