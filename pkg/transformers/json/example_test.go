// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package json_test

import (
	"encoding/json"
	"fmt"

	mgjson "github.com/absmach/magistrala/pkg/transformers/json"
)

func ExampleParseFlat() {
	in := map[string]interface{}{
		"key1":                 "value1",
		"key2":                 "value2",
		"key5/nested1/nested2": "value3",
		"key5/nested1/nested3": "value4",
		"key5/nested2/nested4": "value5",
	}

	out := mgjson.ParseFlat(in)
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
	in := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key5": map[string]interface{}{
			"nested1": map[string]interface{}{
				"nested2": "value3",
				"nested3": "value4",
			},
			"nested2": map[string]interface{}{
				"nested4": "value5",
			},
		},
	}
	out, err := mgjson.Flatten(in)
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
