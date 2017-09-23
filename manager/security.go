package manager

// Hasher specifies an API for generating hashes of an arbitrary textual
// content.
type Hasher interface {
	// Hash generates the hashed string from plain-text.
	Hash(string) (string, error)

	// Compare compares plain-text version to the hashed one. An error should
	// indicate failed comparison.
	Compare(string, string) error
}

// IdentityProvider specifies an API for identity management via security
// tokens.
type IdentityProvider interface {
	// TemporaryKey generates the temporary access token.
	TemporaryKey(string) (string, error)

	// PermanentKey generates the non-expiring access token.
	PermanentKey(string) (string, error)

	// Identity extracts the entity identifier given its secret key.
	Identity(string) (string, error)
}
