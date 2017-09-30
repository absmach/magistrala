package manager

var _ Service = (*managerService)(nil)

type managerService struct {
	users    UserRepository
	clients  ClientRepository
	channels ChannelRepository
	hasher   Hasher
	idp      IdentityProvider
}

// NewService instantiates the domain service implementation.
func NewService(users UserRepository, clients ClientRepository, channels ChannelRepository,
	hasher Hasher, idp IdentityProvider) Service {
	return &managerService{
		users:    users,
		clients:  clients,
		channels: channels,
		hasher:   hasher,
		idp:      idp,
	}
}

func (ms *managerService) Register(user User) error {
	hash, err := ms.hasher.Hash(user.Password)
	if err != nil {
		return ErrMalformedEntity
	}

	user.Password = hash
	return ms.users.Save(user)
}

func (ms *managerService) Login(user User) (string, error) {
	dbUser, err := ms.users.One(user.Email)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	if err := ms.hasher.Compare(user.Password, dbUser.Password); err != nil {
		return "", ErrUnauthorizedAccess
	}

	return ms.idp.TemporaryKey(user.Email)
}

func (ms *managerService) Identity(key string) (string, error) {
	client, err := ms.idp.Identity(key)
	if err != nil {
		return "", err
	}

	return client, nil
}

func (ms *managerService) AddClient(key string, client Client) (string, error) {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return "", err
	}

	if _, err := ms.users.One(sub); err != nil {
		return "", ErrUnauthorizedAccess
	}

	client.ID = ms.clients.Id()
	client.Owner = sub
	client.Key, _ = ms.idp.PermanentKey(client.ID)

	return client.ID, ms.clients.Save(client)
}

func (ms *managerService) UpdateClient(key string, client Client) error {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return err
	}

	if _, err := ms.users.One(sub); err != nil {
		return ErrUnauthorizedAccess
	}

	client.Owner = sub

	return ms.clients.Update(client)
}

func (ms *managerService) ViewClient(key, id string) (Client, error) {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return Client{}, err
	}

	if _, err := ms.users.One(sub); err != nil {
		return Client{}, ErrUnauthorizedAccess
	}

	return ms.clients.One(sub, id)
}

func (ms *managerService) ListClients(key string) ([]Client, error) {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return nil, err
	}

	if _, err := ms.users.One(sub); err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ms.clients.All(sub), nil
}

func (ms *managerService) RemoveClient(key, id string) error {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return err
	}

	if _, err := ms.users.One(sub); err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.clients.Remove(sub, id)
}

func (ms *managerService) CreateChannel(key string, channel Channel) (string, error) {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return "", err
	}

	if _, err := ms.users.One(sub); err != nil {
		return "", ErrUnauthorizedAccess
	}

	channel.Owner = sub
	return ms.channels.Save(channel)
}

func (ms *managerService) UpdateChannel(key string, channel Channel) error {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return err
	}

	if _, err := ms.users.One(sub); err != nil {
		return ErrUnauthorizedAccess
	}

	channel.Owner = sub
	return ms.channels.Update(channel)
}

func (ms *managerService) ViewChannel(key, id string) (Channel, error) {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return Channel{}, err
	}

	if _, err := ms.users.One(sub); err != nil {
		return Channel{}, ErrUnauthorizedAccess
	}

	return ms.channels.One(sub, id)
}

func (ms *managerService) ListChannels(key string) ([]Channel, error) {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return nil, err
	}

	if _, err := ms.users.One(sub); err != nil {
		return nil, ErrUnauthorizedAccess
	}

	return ms.channels.All(sub), nil
}

func (ms *managerService) RemoveChannel(key, id string) error {
	sub, err := ms.idp.Identity(key)
	if err != nil {
		return err
	}

	if _, err := ms.users.One(sub); err != nil {
		return ErrUnauthorizedAccess
	}

	return ms.channels.Remove(sub, id)
}

func (ms *managerService) CanAccess(key, channel string) bool {
	client, err := ms.idp.Identity(key)
	if err != nil {
		return false
	}

	return ms.channels.HasClient(channel, client)
}
