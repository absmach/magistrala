/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package controllers

import (
	"fmt"
	"log"
	"os"

	"github.com/xeipuuv/gojsonschema"
)

/**
 * Function validates JSON schema for `device` od `channel` models
 * By convention, Schema files must be kept as:
 * - ./models/deviceSchema.json
 * - ./models/channelSchema.json
 */
func validateJsonSchema(model string, body map[string]interface{}) bool {
	pwd, _ := os.Getwd()
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + pwd +
		"/models/" + model + "Schema.json")
	bodyLoader := gojsonschema.NewGoLoader(body)
	result, err := gojsonschema.Validate(schemaLoader, bodyLoader)
	if err != nil {
		log.Print(err.Error())
	}

	if result.Valid() {
		fmt.Printf("The document is valid\n")
		return true
	} else {
		fmt.Printf("The document is not valid. See errors :\n")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
		return false
	}
}
