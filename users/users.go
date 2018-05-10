package users

var _ Service = (*usersService)(nil)

type usersService struct {
	users  UserRepository
	hasher Hasher
	idp    IdentityProvider
}

// New instantiates the users service implementation.
func New(users UserRepository, hasher Hasher, idp IdentityProvider) Service {
	return &usersService{users, hasher, idp}
}

func (svc usersService) Register(user User) error {
	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return ErrMalformedEntity
	}

	user.Password = hash
	return svc.users.Save(user)
}

func (svc usersService) Login(user User) (string, error) {
	dbUser, err := svc.users.One(user.Email)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	if err := svc.hasher.Compare(user.Password, dbUser.Password); err != nil {
		return "", ErrUnauthorizedAccess
	}

	return svc.idp.TemporaryKey(user.Email)
}

func (svc usersService) Identify(token string) (string, error) {
	id, err := svc.idp.Identity(token)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}
	return id, nil
}
