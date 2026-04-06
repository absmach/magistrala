// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"encoding/json"
	"fmt"

	smqjson "github.com/absmach/magistrala/pkg/transformers/json"
)

func ExampleParseFlat() {
	in := map[string]any{
		"key1":                 "value1",
		"key2":                 "value2",
		"key5/nested1/nested2": "value3",
		"key5/nested1/nested3": "value4",
		"key5/nested2/nested4": "value5",
	}

	out := smqjson.ParseFlat(in)
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:{
	//   "key1": "value1",
	//   "key2": "value2",
	//   "key5": {
	//     "nested1": {
	//       "nested2": "value3",
	//       "nested3": "value4"
	//     },
	//     "nested2": {
	//       "nested4": "value5"
	//     }
	//   }
	// }
}

func ExampleFlatten() {
	in := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key5": map[string]any{
			"nested1": map[string]any{
				"nested2": "value3",
				"nested3": "value4",
			},
			"nested2": map[string]any{
				"nested4": "value5",
			},
		},
	}
	out, err := smqjson.Flatten(in)
	if err != nil {
		panic(err)
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output:{
	//   "key1": "value1",
	//   "key2": "value2",
	//   "key5/nested1/nested2": "value3",
	//   "key5/nested1/nested3": "value4",
	//   "key5/nested2/nested4": "value5"
	// }
}
