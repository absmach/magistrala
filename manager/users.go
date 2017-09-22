package manager

// User represents a Mainflux user account. Each user is identified given its
// email and password.
type User struct {
	Email    string
	Password string
}

func (u *User) validate() error {
	if u.Email == "" || u.Password == "" {
		return ErrInvalidCredentials
	}

	return nil
}

// UserRepository specifies an account persistence API.
type UserRepository interface {
	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(User) error

	// One retrieves user by its unique identifier (i.e. email).
	One(string) (User, error)
}
