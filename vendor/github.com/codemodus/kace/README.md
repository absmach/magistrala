# kace

    go get "github.com/codemodus/kace"

Package kace provides common case conversion functions which take into 
consideration common initialisms.

## Usage

```go
func Camel(s string) string
func Kebab(s string) string
func KebabUpper(s string) string
func Pascal(s string) string
func Snake(s string) string
func SnakeUpper(s string) string
type Kace
    func New(initialisms map[string]bool) (*Kace, error)
    func (k *Kace) Camel(s string) string
    func (k *Kace) Kebab(s string) string
    func (k *Kace) KebabUpper(s string) string
    func (k *Kace) Pascal(s string) string
    func (k *Kace) Snake(s string) string
    func (k *Kace) SnakeUpper(s string) string
```

### Setup

```go
import (
    "fmt"

    "github.com/codemodus/kace"
)

func main() {
    s := "this is a test sql."

    fmt.Println(kace.Camel(s))
    fmt.Println(kace.Pascal(s))

    fmt.Println(kace.Snake(s))
    fmt.Println(kace.SnakeUpper(s))

    fmt.Println(kace.Kebab(s))
    fmt.Println(kace.KebabUpper(s))

    customInitialisms := map[string]bool{
        "THIS": true,
    }
    k, err := kace.New(customInitialisms)
    if err != nil {
        // handle error
    }

    fmt.Println(k.Camel(s))
    fmt.Println(k.Pascal(s))

    fmt.Println(k.Snake(s))
    fmt.Println(k.SnakeUpper(s))

    fmt.Println(k.Kebab(s))
    fmt.Println(k.KebabUpper(s))

    // Output:
    // thisIsATestSQL
    // ThisIsATestSQL
    // this_is_a_test_sql
    // THIS_IS_A_TEST_SQL
    // this-is-a-test-sql
    // THIS-IS-A-TEST-SQL
    // thisIsATestSql
    // THISIsATestSql
    // this_is_a_test_sql
    // THIS_IS_A_TEST_SQL
    // this-is-a-test-sql
    // THIS-IS-A-TEST-SQL
}
```

## More Info

### TODO

#### Test Trie

 Test the current trie.

## Documentation

View the [GoDoc](http://godoc.org/github.com/codemodus/kace)

## Benchmarks

    benchmark                 iter       time/iter   bytes alloc        allocs
    ---------                 ----       ---------   -----------        ------
    BenchmarkCamel4        2000000    947.00 ns/op      112 B/op   3 allocs/op
    BenchmarkSnake4        2000000    696.00 ns/op      128 B/op   2 allocs/op
    BenchmarkSnakeUpper4   2000000    679.00 ns/op      128 B/op   2 allocs/op
    BenchmarkKebab4        2000000    691.00 ns/op      128 B/op   2 allocs/op
    BenchmarkKebabUpper4   2000000    677.00 ns/op      128 B/op   2 allocs/op
