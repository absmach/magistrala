package mocks

import "fmt"

func key(owner, id string) string {
	return fmt.Sprintf("%s-%s", owner, id)
}
