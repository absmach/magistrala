package fn

type FuncList []func()

// Return a function that executions all added functions
//
// Functions are executed in reverse order they were added.
func (c FuncList) ToFunction() func() {
	return func() {
		for i := range c {
			c[len(c)-1-i]()
		}
	}
}

// Execute all added functions
func (c FuncList) Execute() {
	c.ToFunction()()
}
