# GoSenML

Go (golang) library for [SenML](https://tools.ietf.org/id/draft-jennings-core-senml-00.txt)

# TODO
- [ ] XML encoder
- [ ] Binary (exi/cbor) encoder
- [ ] Message 'compaction'
- [ ] Proper message validation

# Example

```go
package main

import (
	"fmt"
	"github.com/krylovsk/gosenml"
)

func main() {
	e := gosenml.Entry{
		Name:  "sensor1",
		Units: "degC",
	}
	v := 42.0
	e.Value = &v

	m1 := gosenml.NewMessage(e)
	m1.BaseName = "http://example.com/"

	encoder := gosenml.NewJSONEncoder()
	decoder := gosenml.NewJSONDecoder()

	b, _ := encoder.EncodeMessage(m1)
	fmt.Println(string(b))

	m2, _ := decoder.DecodeMessage(b)

	m3 := m2.Expand()
	b, _ = encoder.EncodeMessage(&m3)
	fmt.Println(string(b))
}

```
