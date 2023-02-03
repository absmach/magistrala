package env

// NewConfig gets configuration from environment variable
func NewConfig[T any](opts ...Options) (T, error) {
	var cfg T
	if err := Parse(&cfg, opts...); err != nil {
		return cfg, err
	}
	return cfg, nil
}
