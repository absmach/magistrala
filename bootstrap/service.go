// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"

	"github.com/absmach/magistrala"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
)

var (
	// ErrExternalKey indicates a non-existent bootstrap configuration for given external key.
	ErrExternalKey = errors.NewAuthZError("failed to get bootstrap configuration for given external key")

	// ErrExternalKeySecure indicates error in getting bootstrap configuration for given encrypted external key.
	ErrExternalKeySecure = errors.NewAuthZError("failed to get bootstrap configuration for given encrypted external key")

	// ErrBootstrap indicates error in getting bootstrap configuration.
	ErrBootstrap = errors.New("failed to read bootstrap configuration")

	// ErrAddBootstrap indicates error in adding bootstrap configuration.
	ErrAddBootstrap = errors.NewServiceError("failed to add bootstrap configuration")

	// ErrBootstrapStatus indicates an invalid bootstrap status.
	ErrBootstrapStatus = errors.NewRequestError("invalid bootstrap status")

	errRemoveBootstrap = errors.New("failed to remove bootstrap configuration")
	errEnableConfig    = errors.New("failed to enable bootstrap configuration")
	errDisableConfig   = errors.New("failed to disable bootstrap configuration")
	errRemoveConfig    = errors.New("failed to remove bootstrap configuration")
	errUpdateCert      = errors.New("failed to update cert")

	errCreateProfile   = errors.New("failed to create profile")
	errViewProfile     = errors.New("failed to view profile")
	errUpdateProfile   = errors.New("failed to update profile")
	errDeleteProfile   = errors.New("failed to delete profile")
	errListProfiles    = errors.New("failed to list profiles")
	errAssignProfile   = errors.New("failed to assign profile to enrollment")
	errBindResources   = errors.New("failed to bind resources")
	errListBindings    = errors.New("failed to list bindings")
	errRefreshBinding  = errors.New("failed to refresh bindings")
	errRenderBootstrap = errors.New("failed to render bootstrap configuration")
)

var _ Service = (*bootstrapService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Add adds new Client Config to the user identified by the provided token.
	Add(ctx context.Context, session smqauthn.Session, token string, cfg Config) (Config, error)

	// View returns Client Config with given ID belonging to the user identified by the given token.
	View(ctx context.Context, session smqauthn.Session, id string) (Config, error)

	// Update updates editable fields of the provided Config.
	Update(ctx context.Context, session smqauthn.Session, cfg Config) error

	// UpdateCert updates an existing Config certificate and token.
	// A non-nil error is returned to indicate operation failure.
	UpdateCert(ctx context.Context, session smqauthn.Session, clientID, clientCert, clientKey, caCert string) (Config, error)

	// List returns subset of Configs with given search params that belong to the
	// user identified by the given token.
	List(ctx context.Context, session smqauthn.Session, filter Filter, offset, limit uint64) (ConfigsPage, error)

	// Remove removes Config with specified token that belongs to the user identified by the given token.
	Remove(ctx context.Context, session smqauthn.Session, id string) error

	// Bootstrap returns Config to the Client with provided external ID using external key.
	Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (Config, error)

	// EnableConfig enables the Config so its device can successfully bootstrap.
	EnableConfig(ctx context.Context, session smqauthn.Session, id string) (Config, error)

	// DisableConfig disables the Config, preventing its device from bootstrapping.
	DisableConfig(ctx context.Context, session smqauthn.Session, id string) (Config, error)

	// Methods RemoveConfig, UpdateChannel, and RemoveChannel are used as
	// handlers for events. That's why these methods surpass ownership check.

	// RemoveConfigHandler removes Configuration with id received from an event.
	RemoveConfigHandler(ctx context.Context, id string) error

	// CreateProfile persists a new device Profile.
	CreateProfile(ctx context.Context, session smqauthn.Session, p Profile) (Profile, error)

	// ViewProfile returns the Profile with the given ID.
	ViewProfile(ctx context.Context, session smqauthn.Session, profileID string) (Profile, error)

	// UpdateProfile updates editable fields of the given Profile.
	UpdateProfile(ctx context.Context, session smqauthn.Session, p Profile) error

	// ListProfiles returns a page of Profiles belonging to the domain.
	ListProfiles(ctx context.Context, session smqauthn.Session, offset, limit uint64) (ProfilesPage, error)

	// DeleteProfile removes the Profile with the given ID.
	DeleteProfile(ctx context.Context, session smqauthn.Session, profileID string) error

	// AssignProfile sets the ProfileID on an existing enrollment (Config).
	AssignProfile(ctx context.Context, session smqauthn.Session, configID, profileID string) error

	// BindResources resolves the requested bindings through their owning services,
	// stores snapshots, and marks the enrollment renderable when all required slots
	// are satisfied.
	BindResources(ctx context.Context, session smqauthn.Session, token, configID string, bindings []BindingRequest) error

	// ListBindings returns all stored binding snapshots for an enrollment.
	ListBindings(ctx context.Context, session smqauthn.Session, configID string) ([]BindingSnapshot, error)

	// RefreshBindings re-resolves all existing bindings for an enrollment and
	// updates the stored snapshots.
	RefreshBindings(ctx context.Context, session smqauthn.Session, token, configID string) error
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
type ConfigReader interface {
	ReadConfig(Config, bool) (any, error)
}

type bootstrapService struct {
	policies   policies.Service
	configs    ConfigRepository
	profiles   ProfileRepository
	bindings   BindingStore
	resolver   BindingResolver
	renderer   Renderer
	hasher     Hasher
	sdk        mgsdk.SDK
	encKey     []byte
	idProvider magistrala.IDProvider
}

// New returns new Bootstrap service.
func New(policyService policies.Service, configs ConfigRepository, sdk mgsdk.SDK, hasher Hasher, encKey []byte, idp magistrala.IDProvider) Service {
	return &bootstrapService{
		configs:    configs,
		hasher:     hasher,
		sdk:        sdk,
		policies:   policyService,
		encKey:     encKey,
		idProvider: idp,
	}
}

// NewWithProfiles returns a Bootstrap service with profile and binding support enabled.
func NewWithProfiles(
	policyService policies.Service,
	configs ConfigRepository,
	profiles ProfileRepository,
	bindings BindingStore,
	resolver BindingResolver,
	renderer Renderer,
	sdk mgsdk.SDK,
	hasher Hasher,
	encKey []byte,
	idp magistrala.IDProvider,
) Service {
	return &bootstrapService{
		configs:    configs,
		profiles:   profiles,
		bindings:   bindings,
		resolver:   resolver,
		renderer:   renderer,
		hasher:     hasher,
		sdk:        sdk,
		policies:   policyService,
		encKey:     encKey,
		idProvider: idp,
	}
}

func (bs bootstrapService) Add(ctx context.Context, session smqauthn.Session, token string, cfg Config) (Config, error) {
	id, err := bs.idProvider.ID()
	if err != nil {
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}

	hashedKey, err := bs.hasher.Hash(cfg.ExternalKey)
	if err != nil {
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}

	cfg.ID = id
	cfg.DomainID = session.DomainID
	cfg.Status = DisabledStatus
	cfg.ExternalKey = hashedKey

	saved, err := bs.configs.Save(ctx, cfg)
	if err != nil {
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}

	cfg.ID = saved
	return cfg, nil
}

func (bs bootstrapService) View(ctx context.Context, session smqauthn.Session, id string) (Config, error) {
	cfg, err := bs.configs.RetrieveByID(ctx, session.DomainID, id)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return cfg, nil
}

func (bs bootstrapService) Update(ctx context.Context, session smqauthn.Session, cfg Config) error {
	cfg.DomainID = session.DomainID
	if err := bs.configs.Update(ctx, cfg); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (bs bootstrapService) UpdateCert(ctx context.Context, session smqauthn.Session, clientID, clientCert, clientKey, caCert string) (Config, error) {
	cfg, err := bs.configs.UpdateCert(ctx, session.DomainID, clientID, clientCert, clientKey, caCert)
	if err != nil {
		return Config{}, errors.Wrap(errUpdateCert, err)
	}
	return cfg, nil
}

func (bs bootstrapService) List(ctx context.Context, session smqauthn.Session, filter Filter, offset, limit uint64) (ConfigsPage, error) {
	return bs.configs.RetrieveAll(ctx, session.DomainID, []string{}, filter, offset, limit), nil
}

func (bs bootstrapService) Remove(ctx context.Context, session smqauthn.Session, id string) error {
	if err := bs.configs.Remove(ctx, session.DomainID, id); err != nil {
		return errors.Wrap(errRemoveBootstrap, err)
	}
	return nil
}

func (bs bootstrapService) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (Config, error) {
	cfg, err := bs.configs.RetrieveByExternalID(ctx, externalID)
	if err != nil {
		return cfg, errors.Wrap(ErrBootstrap, err)
	}
	if secure {
		dec, err := bs.dec(externalKey)
		if err != nil {
			return Config{}, errors.Wrap(ErrExternalKeySecure, err)
		}
		externalKey = dec
	}

	if err := bs.hasher.Compare(externalKey, cfg.ExternalKey); err != nil {
		return Config{}, ErrExternalKey
	}
	if cfg.Status == DisabledStatus {
		return Config{}, ErrBootstrap
	}

	cfg, err = bs.renderBootstrapConfig(ctx, cfg)
	if err != nil {
		return Config{}, errors.Wrap(ErrBootstrap, err)
	}

	return cfg, nil
}

func (bs bootstrapService) renderBootstrapConfig(ctx context.Context, cfg Config) (Config, error) {
	if cfg.ProfileID == "" {
		return cfg, nil
	}
	if bs.profiles == nil || bs.bindings == nil || bs.renderer == nil {
		return Config{}, errors.Wrap(errRenderBootstrap, errors.New("profile rendering support not configured"))
	}

	profile, err := bs.profiles.RetrieveByID(ctx, cfg.DomainID, cfg.ProfileID)
	if err != nil {
		return Config{}, errors.Wrap(errRenderBootstrap, err)
	}

	bindings, err := bs.bindings.Retrieve(ctx, cfg.ID)
	if err != nil {
		return Config{}, errors.Wrap(errRenderBootstrap, err)
	}
	if err := validateRequiredBindings(profile, bindings); err != nil {
		return Config{}, errors.Wrap(errRenderBootstrap, err)
	}
	bindings, err = bs.decryptSecretSnapshots(bindings)
	if err != nil {
		return Config{}, errors.Wrap(errRenderBootstrap, err)
	}

	rendered, err := bs.renderer.Render(profile, cfg, bindings)
	if err != nil {
		return Config{}, errors.Wrap(errRenderBootstrap, err)
	}

	cfg.Content = string(rendered)
	return cfg, nil
}

func (bs bootstrapService) EnableConfig(ctx context.Context, session smqauthn.Session, id string) (Config, error) {
	cfg, err := bs.changeConfigStatus(ctx, session.DomainID, id, EnabledStatus)
	if err != nil {
		return Config{}, errors.Wrap(errEnableConfig, err)
	}
	return cfg, nil
}

func (bs bootstrapService) DisableConfig(ctx context.Context, session smqauthn.Session, id string) (Config, error) {
	cfg, err := bs.changeConfigStatus(ctx, session.DomainID, id, DisabledStatus)
	if err != nil {
		return Config{}, errors.Wrap(errDisableConfig, err)
	}
	return cfg, nil
}

func (bs bootstrapService) changeConfigStatus(ctx context.Context, domainID, id string, status Status) (Config, error) {
	cfg, err := bs.configs.RetrieveByID(ctx, domainID, id)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if cfg.Status == status {
		return Config{}, svcerr.ErrStatusAlreadyAssigned
	}
	if err := bs.configs.ChangeStatus(ctx, domainID, id, status); err != nil {
		return Config{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	cfg.Status = status
	return cfg, nil
}

func (bs bootstrapService) RemoveConfigHandler(ctx context.Context, id string) error {
	if err := bs.configs.RemoveClient(ctx, id); err != nil {
		return errors.Wrap(errRemoveConfig, err)
	}
	return nil
}

// --- Profile management ---

func (bs bootstrapService) CreateProfile(ctx context.Context, session smqauthn.Session, p Profile) (Profile, error) {
	if bs.profiles == nil {
		return Profile{}, errors.Wrap(errCreateProfile, errors.New("profile repository not configured"))
	}
	id, err := bs.idProvider.ID()
	if err != nil {
		return Profile{}, errors.Wrap(errCreateProfile, err)
	}
	p.ID = id
	p.DomainID = session.DomainID
	if p.TemplateFormat == "" {
		p.TemplateFormat = TemplateFormatGoTemplate
	}
	if p.Version == 0 {
		p.Version = 1
	}
	if err := validateProfileBindingSlots(p); err != nil {
		return Profile{}, errors.Wrap(errCreateProfile, err)
	}
	saved, err := bs.profiles.Save(ctx, p)
	if err != nil {
		return Profile{}, errors.Wrap(errCreateProfile, err)
	}
	return saved, nil
}

func (bs bootstrapService) ViewProfile(ctx context.Context, session smqauthn.Session, profileID string) (Profile, error) {
	if bs.profiles == nil {
		return Profile{}, errors.Wrap(errViewProfile, errors.New("profile repository not configured"))
	}
	p, err := bs.profiles.RetrieveByID(ctx, session.DomainID, profileID)
	if err != nil {
		return Profile{}, errors.Wrap(errViewProfile, err)
	}
	return p, nil
}

func (bs bootstrapService) UpdateProfile(ctx context.Context, session smqauthn.Session, p Profile) error {
	if bs.profiles == nil {
		return errors.Wrap(errUpdateProfile, errors.New("profile repository not configured"))
	}
	p.DomainID = session.DomainID
	if p.TemplateFormat == "" {
		p.TemplateFormat = TemplateFormatGoTemplate
	}
	if err := validateProfileBindingSlots(p); err != nil {
		return errors.Wrap(errUpdateProfile, err)
	}
	if err := bs.profiles.Update(ctx, p); err != nil {
		return errors.Wrap(errUpdateProfile, err)
	}
	return nil
}

func (bs bootstrapService) ListProfiles(ctx context.Context, session smqauthn.Session, offset, limit uint64) (ProfilesPage, error) {
	if bs.profiles == nil {
		return ProfilesPage{}, errors.Wrap(errListProfiles, errors.New("profile repository not configured"))
	}
	page, err := bs.profiles.RetrieveAll(ctx, session.DomainID, offset, limit)
	if err != nil {
		return ProfilesPage{}, errors.Wrap(errListProfiles, err)
	}
	return page, nil
}

func (bs bootstrapService) DeleteProfile(ctx context.Context, session smqauthn.Session, profileID string) error {
	if bs.profiles == nil {
		return errors.Wrap(errDeleteProfile, errors.New("profile repository not configured"))
	}
	if err := bs.profiles.Delete(ctx, session.DomainID, profileID); err != nil {
		return errors.Wrap(errDeleteProfile, err)
	}
	return nil
}

// --- Enrollment-profile assignment ---

func (bs bootstrapService) AssignProfile(ctx context.Context, session smqauthn.Session, configID, profileID string) error {
	if bs.profiles == nil {
		return errors.Wrap(errAssignProfile, errors.New("profile repository not configured"))
	}
	// Validate profile exists in domain.
	if _, err := bs.profiles.RetrieveByID(ctx, session.DomainID, profileID); err != nil {
		return errors.Wrap(errAssignProfile, err)
	}
	if err := bs.configs.AssignProfile(ctx, session.DomainID, configID, profileID); err != nil {
		return errors.Wrap(errAssignProfile, err)
	}
	return nil
}

// --- Binding management ---

func (bs bootstrapService) BindResources(ctx context.Context, session smqauthn.Session, token, configID string, requested []BindingRequest) error {
	if bs.profiles == nil || bs.bindings == nil || bs.resolver == nil {
		return errors.Wrap(errBindResources, errors.New("binding support not configured"))
	}
	cfg, err := bs.configs.RetrieveByID(ctx, session.DomainID, configID)
	if err != nil {
		return errors.Wrap(errBindResources, err)
	}
	profile, err := bs.profiles.RetrieveByID(ctx, session.DomainID, cfg.ProfileID)
	if err != nil {
		return errors.Wrap(errBindResources, err)
	}
	if err := validateRequestedBindings(profile, requested); err != nil {
		return errors.Wrap(errBindResources, err)
	}
	snapshots, err := bs.resolver.Resolve(ctx, ResolveRequest{
		Enrollment: cfg,
		Token:      token,
		Requested:  requested,
	})
	if err != nil {
		return errors.Wrap(errBindResources, err)
	}
	existing, err := bs.bindings.Retrieve(ctx, configID)
	if err != nil {
		return errors.Wrap(errBindResources, err)
	}
	if err := validateRequiredBindings(profile, mergeBindingSnapshots(existing, snapshots)); err != nil {
		return errors.Wrap(errBindResources, err)
	}
	snapshots, err = bs.encryptSecretSnapshots(snapshots)
	if err != nil {
		return errors.Wrap(errBindResources, err)
	}
	return bs.bindings.Save(ctx, configID, snapshots)
}

func (bs bootstrapService) ListBindings(ctx context.Context, session smqauthn.Session, configID string) ([]BindingSnapshot, error) {
	if bs.bindings == nil {
		return nil, errors.Wrap(errListBindings, errors.New("binding support not configured"))
	}
	if _, err := bs.configs.RetrieveByID(ctx, session.DomainID, configID); err != nil {
		return nil, errors.Wrap(errListBindings, err)
	}
	snapshots, err := bs.bindings.Retrieve(ctx, configID)
	if err != nil {
		return nil, errors.Wrap(errListBindings, err)
	}
	return hideSecretSnapshots(snapshots), nil
}

func (bs bootstrapService) RefreshBindings(ctx context.Context, session smqauthn.Session, token, configID string) error {
	if bs.profiles == nil || bs.bindings == nil || bs.resolver == nil {
		return errors.Wrap(errRefreshBinding, errors.New("binding support not configured"))
	}
	cfg, err := bs.configs.RetrieveByID(ctx, session.DomainID, configID)
	if err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	profile, err := bs.profiles.RetrieveByID(ctx, session.DomainID, cfg.ProfileID)
	if err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	existing, err := bs.bindings.Retrieve(ctx, configID)
	if err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	if len(existing) == 0 {
		return nil
	}
	// Re-resolve every existing binding to refresh its snapshot.
	requested := make([]BindingRequest, len(existing))
	for i, b := range existing {
		requested[i] = BindingRequest{Slot: b.Slot, Type: b.Type, ResourceID: b.ResourceID}
	}
	if err := validateRequestedBindings(profile, requested); err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	refreshed, err := bs.resolver.Resolve(ctx, ResolveRequest{
		Enrollment: cfg,
		Token:      token,
		Requested:  requested,
	})
	if err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	if err := validateRequiredBindings(profile, refreshed); err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	refreshed, err = bs.encryptSecretSnapshots(refreshed)
	if err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	return bs.bindings.Save(ctx, configID, refreshed)
}

func (bs bootstrapService) dec(in string) (string, error) {
	ciphertext, err := hex.DecodeString(in)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(bs.encKey)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aes.BlockSize {
		return "", err
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext), nil
}
