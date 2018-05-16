package things

// IdentityProvider specifies an API for generating unique identifiers.
type IdentityProvider interface {
	// ID generates the unique identifier.
	ID() string
}
