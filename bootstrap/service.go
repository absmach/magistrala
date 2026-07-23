// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/absmach/magistrala"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
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

	// ErrExternalKeyUnavailable indicates that an enrollment has no recoverable external key.
	ErrExternalKeyUnavailable = errors.NewRequestError("external key is not available")

	// ErrBootstrapDisabled indicates that secure credentials cannot be generated for a disabled enrollment.
	ErrBootstrapDisabled = errors.NewRequestError("bootstrap configuration is disabled")

	errRemoveBootstrap = errors.New("failed to remove bootstrap configuration")
	errEnableConfig    = errors.New("failed to enable bootstrap configuration")
	errDisableConfig   = errors.New("failed to disable bootstrap configuration")
	errUpdateCert      = errors.New("failed to update cert")

	errCreateProfile            = errors.New("failed to create profile")
	errViewProfile              = errors.New("failed to view profile")
	errUpdateProfile            = errors.New("failed to update profile")
	errDeleteProfile            = errors.New("failed to delete profile")
	errListProfiles             = errors.New("failed to list profiles")
	errAssignProfile            = errors.New("failed to assign profile to enrollment")
	errBindResources            = errors.New("failed to bind resources")
	errListBindings             = errors.New("failed to list bindings")
	errRefreshBinding           = errors.New("failed to refresh bindings")
	errRenderBootstrap          = errors.New("failed to render bootstrap configuration")
	errCreateTransportKey       = errors.New("failed to create domain bootstrap transport key")
	errViewTransportKey         = errors.New("failed to view domain bootstrap transport key")
	errRevealTransportKey       = errors.New("failed to reveal domain bootstrap transport key")
	errRotateTransportKey       = errors.New("failed to rotate domain bootstrap transport key")
	errGenerateSecureCredential = errors.New("failed to generate secure bootstrap credential")
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
	UpdateCert(ctx context.Context, session smqauthn.Session, id, clientCert, clientKey, caCert string) (Config, error)

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

	// CreateProfile persists a new device Profile.
	CreateProfile(ctx context.Context, session smqauthn.Session, p Profile) (Profile, error)

	// ViewProfile returns the Profile with the given ID.
	ViewProfile(ctx context.Context, session smqauthn.Session, profileID string) (Profile, error)

	// UpdateProfile updates editable fields of the given Profile and returns the updated Profile.
	UpdateProfile(ctx context.Context, session smqauthn.Session, p Profile) (Profile, error)

	// ListProfiles returns a page of Profiles belonging to the domain.
	ListProfiles(ctx context.Context, session smqauthn.Session, offset, limit uint64, name string) (ProfilesPage, error)

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

	CreateDomainTransportKey(ctx context.Context, session smqauthn.Session) (DomainTransportKey, error)
	ViewDomainTransportKey(ctx context.Context, session smqauthn.Session) (DomainTransportKey, error)
	RevealDomainTransportKey(ctx context.Context, session smqauthn.Session, keyID string) (DomainTransportKey, error)
	RotateDomainTransportKey(ctx context.Context, session smqauthn.Session) (DomainTransportKey, error)
	GenerateSecureCredential(ctx context.Context, session smqauthn.Session, configID string) (SecureBootstrapCredential, error)
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
type ConfigReader interface {
	ReadConfig(Config, bool) (any, error)
}

type bootstrapService struct {
	configs       ConfigRepository
	profiles      ProfileRepository
	bindings      BindingStore
	transportKeys DomainTransportKeyRepository
	resolver      BindingResolver
	renderer      Renderer
	sdk           mgsdk.SDK
	dbCipher      *SecretCipher
	idProvider    magistrala.IDProvider
	now           func() time.Time
}

// New returns new Bootstrap service.
func New(
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
	return NewWithTransportKeys(configs, profiles, bindings, nil, resolver, renderer, sdk, encKey, "primary", idp)
}

// NewWithTransportKeys returns a Bootstrap service with per-domain encrypted
// device transport key support.
func NewWithTransportKeys(
	configs ConfigRepository,
	profiles ProfileRepository,
	bindings BindingStore,
	transportKeys DomainTransportKeyRepository,
	resolver BindingResolver,
	renderer Renderer,
	sdk mgsdk.SDK,
	encKey []byte,
	encKeyID string,
	idp magistrala.IDProvider,
) Service {
	dbCipher, _ := NewSecretCipher(encKey, encKeyID)
	return &bootstrapService{
		configs:       configs,
		profiles:      profiles,
		bindings:      bindings,
		transportKeys: transportKeys,
		resolver:      resolver,
		renderer:      renderer,
		sdk:           sdk,
		dbCipher:      dbCipher,
		idProvider:    idp,
		now:           time.Now,
	}
}

func (bs bootstrapService) Add(ctx context.Context, session smqauthn.Session, token string, cfg Config) (Config, error) {
	id, err := bs.idProvider.ID()
	if err != nil {
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}

	cfg.ID = id
	cfg.DomainID = session.DomainID
	cfg.Status = Active
	if bs.dbCipher == nil {
		return Config{}, errors.Wrap(ErrAddBootstrap, errors.New("database encryption key is invalid"))
	}
	encryptedKey, err := bs.dbCipher.seal("config-external-key", []byte(cfg.ExternalKey), configSecretAAD(cfg))
	if err != nil {
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}
	stored := cfg
	stored.ExternalKey = encryptedKey

	saved, err := bs.configs.Save(ctx, stored)
	if err != nil {
		if errors.Contains(err, repoerr.ErrConflict) {
			return Config{}, errors.Wrap(svcerr.ErrConflict, err)
		}
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
	return bs.decryptConfigExternalKeyForManagement(cfg, svcerr.ErrViewEntity)
}

func (bs bootstrapService) Update(ctx context.Context, session smqauthn.Session, cfg Config) error {
	cfg.DomainID = session.DomainID
	if cfg.ExternalKey != "" {
		stored, err := bs.configs.RetrieveByID(ctx, session.DomainID, cfg.ID)
		if err != nil {
			return errors.Wrap(svcerr.ErrUpdateEntity, err)
		}
		if bs.dbCipher == nil {
			return errors.Wrap(svcerr.ErrUpdateEntity, errors.New("database encryption key is invalid"))
		}
		cfg.ExternalID = stored.ExternalID
		encryptedKey, err := bs.dbCipher.seal("config-external-key", []byte(cfg.ExternalKey), configSecretAAD(cfg))
		if err != nil {
			return errors.Wrap(svcerr.ErrUpdateEntity, err)
		}
		cfg.ExternalKey = encryptedKey
	}
	if err := bs.configs.Update(ctx, cfg); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (bs bootstrapService) UpdateCert(ctx context.Context, session smqauthn.Session, id, clientCert, clientKey, caCert string) (Config, error) {
	cfg, err := bs.configs.UpdateCert(ctx, session.DomainID, id, clientCert, clientKey, caCert)
	if err != nil {
		return Config{}, errors.Wrap(errUpdateCert, err)
	}
	return cfg, nil
}

func (bs bootstrapService) List(ctx context.Context, session smqauthn.Session, filter Filter, offset, limit uint64) (ConfigsPage, error) {
	page := bs.configs.RetrieveAll(ctx, session.DomainID, filter, offset, limit)
	for i, cfg := range page.Configs {
		decrypted, err := bs.decryptConfigExternalKeyForManagement(cfg, svcerr.ErrViewEntity)
		if err != nil {
			return ConfigsPage{}, err
		}
		page.Configs[i] = decrypted
	}
	return page, nil
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
		dec, err := bs.decryptSecureRequest(ctx, &cfg, externalKey)
		if err != nil {
			return Config{}, errors.Wrap(ErrExternalKeySecure, err)
		}
		externalKey = dec
	}

	decrypted, err := bs.decryptConfigExternalKey(cfg, ErrExternalKey)
	if err != nil {
		return Config{}, err
	}
	if subtle.ConstantTimeCompare([]byte(externalKey), []byte(decrypted.ExternalKey)) != 1 {
		return Config{}, ErrExternalKey
	}
	cfg = decrypted
	if cfg.Status == DisabledStatus {
		return Config{}, ErrBootstrap
	}

	cfg, err = bs.renderBootstrapConfig(ctx, cfg)
	if err != nil {
		return Config{}, errors.Wrap(ErrBootstrap, err)
	}

	return cfg, nil
}

func (bs bootstrapService) decryptConfigExternalKey(cfg Config, outer error) (Config, error) {
	// Plaintext values are accepted only for compatibility with deployments
	// created before at-rest encryption was introduced. Every newly created
	// enrollment is always persisted as a dbv1 envelope. Bcrypt hashes remain
	// intentionally unrecoverable and require enrollment recreation.
	if !strings.HasPrefix(cfg.ExternalKey, databaseEnvelopeVersion+".") {
		if strings.HasPrefix(cfg.ExternalKey, "$2") {
			return Config{}, errors.Wrap(outer, errors.New("legacy hashed external key cannot be recovered; recreate the enrollment"))
		}
		return cfg, nil
	}
	if bs.dbCipher == nil {
		return Config{}, errors.Wrap(outer, errors.New("database encryption key is invalid"))
	}
	plain, err := bs.dbCipher.open("config-external-key", cfg.ExternalKey, configSecretAAD(cfg))
	if err != nil {
		return Config{}, errors.Wrap(outer, err)
	}
	cfg.ExternalKey = string(plain)
	return cfg, nil
}

func (bs bootstrapService) decryptConfigExternalKeyForManagement(cfg Config, outer error) (Config, error) {
	// Bcrypt hashes created by the previous Bootstrap implementation are
	// intentionally one-way. Management reads must still return the enrollment,
	// but cannot reveal a key that no longer exists in recoverable form.
	if strings.HasPrefix(cfg.ExternalKey, "$2") {
		cfg.ExternalKey = ""
		return cfg, nil
	}
	return bs.decryptConfigExternalKey(cfg, outer)
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
	cfg, err = bs.decryptConfigExternalKeyForManagement(cfg, svcerr.ErrViewEntity)
	if err != nil {
		return Config{}, err
	}
	if cfg.Status == status {
		return cfg, nil
	}
	if err := bs.configs.ChangeStatus(ctx, domainID, id, status); err != nil {
		return Config{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	cfg.Status = status
	return cfg, nil
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
	if p.ContentFormat == "" {
		p.ContentFormat = ContentFormatJSON
	}
	p.Version = 1
	if err := validateProfileBindingSlots(p); err != nil {
		return Profile{}, errors.Wrap(errCreateProfile, err)
	}
	if err := validateProfileTemplate(p); err != nil {
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

func (bs bootstrapService) UpdateProfile(ctx context.Context, session smqauthn.Session, p Profile) (Profile, error) {
	if bs.profiles == nil {
		return Profile{}, errors.Wrap(errUpdateProfile, errors.New("profile repository not configured"))
	}
	p.DomainID = session.DomainID
	if err := validateProfileBindingSlots(p); err != nil {
		return Profile{}, errors.Wrap(errUpdateProfile, err)
	}
	if err := validateProfileTemplate(p); err != nil {
		return Profile{}, errors.Wrap(errUpdateProfile, err)
	}
	updated, err := bs.profiles.Update(ctx, p)
	if err != nil {
		return Profile{}, errors.Wrap(errUpdateProfile, err)
	}
	return updated, nil
}

func (bs bootstrapService) ListProfiles(ctx context.Context, session smqauthn.Session, offset, limit uint64, name string) (ProfilesPage, error) {
	if bs.profiles == nil {
		return ProfilesPage{}, errors.Wrap(errListProfiles, errors.New("profile repository not configured"))
	}
	page, err := bs.profiles.RetrieveAll(ctx, session.DomainID, offset, limit, name)
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
	for i := range snapshots {
		snapshots[i].ConfigID = configID
	}
	snapshots, err = bs.encryptSecretSnapshots(snapshots)
	if err != nil {
		return errors.Wrap(errBindResources, err)
	}
	if err := bs.bindings.Save(ctx, configID, snapshots); err != nil {
		return errors.Wrap(errBindResources, err)
	}
	return nil
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
	for i := range refreshed {
		refreshed[i].ConfigID = configID
	}
	refreshed, err = bs.encryptSecretSnapshots(refreshed)
	if err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	return bs.bindings.Save(ctx, configID, refreshed)
}

func (bs bootstrapService) CreateDomainTransportKey(ctx context.Context, session smqauthn.Session) (DomainTransportKey, error) {
	if bs.transportKeys == nil || bs.dbCipher == nil {
		return DomainTransportKey{}, errors.Wrap(errCreateTransportKey, errors.New("domain transport key support not configured"))
	}
	key, err := bs.newDomainTransportKey(session.DomainID)
	if err != nil {
		return DomainTransportKey{}, errors.Wrap(errCreateTransportKey, err)
	}
	if err := bs.transportKeys.Create(ctx, key); err != nil {
		if errors.Contains(err, repoerr.ErrConflict) {
			return DomainTransportKey{}, errors.Wrap(svcerr.ErrConflict, err)
		}
		return DomainTransportKey{}, errors.Wrap(errCreateTransportKey, err)
	}
	return bs.revealTransportKey(key)
}

func (bs bootstrapService) ViewDomainTransportKey(ctx context.Context, session smqauthn.Session) (DomainTransportKey, error) {
	if bs.transportKeys == nil {
		return DomainTransportKey{}, errors.Wrap(errViewTransportKey, errors.New("domain transport key support not configured"))
	}
	key, err := bs.transportKeys.RetrieveCurrent(ctx, session.DomainID)
	if err != nil {
		return DomainTransportKey{}, errors.Wrap(errViewTransportKey, err)
	}
	key.EncryptedSecret = ""
	return key, nil
}

func (bs bootstrapService) RevealDomainTransportKey(ctx context.Context, session smqauthn.Session, keyID string) (DomainTransportKey, error) {
	if bs.transportKeys == nil || bs.dbCipher == nil {
		return DomainTransportKey{}, errors.Wrap(errRevealTransportKey, errors.New("domain transport key support not configured"))
	}
	key, err := bs.transportKeys.Retrieve(ctx, session.DomainID, keyID)
	if err != nil {
		return DomainTransportKey{}, errors.Wrap(errRevealTransportKey, err)
	}
	revealed, err := bs.revealTransportKey(key)
	if err != nil {
		return DomainTransportKey{}, errors.Wrap(errRevealTransportKey, err)
	}
	return revealed, nil
}

func (bs bootstrapService) RotateDomainTransportKey(ctx context.Context, session smqauthn.Session) (DomainTransportKey, error) {
	if bs.transportKeys == nil || bs.dbCipher == nil {
		return DomainTransportKey{}, errors.Wrap(errRotateTransportKey, errors.New("domain transport key support not configured"))
	}
	current, err := bs.transportKeys.RetrieveCurrent(ctx, session.DomainID)
	if err != nil {
		return DomainTransportKey{}, errors.Wrap(errRotateTransportKey, err)
	}
	next, err := bs.newDomainTransportKey(session.DomainID)
	if err != nil {
		return DomainTransportKey{}, errors.Wrap(errRotateTransportKey, err)
	}
	retireAt := bs.now().UTC().Add(24 * time.Hour)
	if err := bs.transportKeys.Rotate(ctx, current.KeyID, next, retireAt); err != nil {
		return DomainTransportKey{}, errors.Wrap(errRotateTransportKey, err)
	}
	revealed, err := bs.revealTransportKey(next)
	if err != nil {
		return DomainTransportKey{}, errors.Wrap(errRotateTransportKey, err)
	}
	return revealed, nil
}

func (bs bootstrapService) newDomainTransportKey(domainID string) (DomainTransportKey, error) {
	keyID, err := bs.idProvider.ID()
	if err != nil {
		return DomainTransportKey{}, err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return DomainTransportKey{}, err
	}
	encrypted, err := bs.dbCipher.seal("domain-transport-key", secret, transportSecretAAD(domainID, keyID))
	if err != nil {
		return DomainTransportKey{}, err
	}
	now := bs.now().UTC()
	return DomainTransportKey{
		DomainID: domainID, KeyID: keyID, EncryptedSecret: encrypted,
		WrappingKeyID: bs.dbCipher.KeyID(), Status: TransportKeyActive,
		Secret: encodeTransportSecret(secret), CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (bs bootstrapService) revealTransportKey(key DomainTransportKey) (DomainTransportKey, error) {
	secret, err := bs.dbCipher.open("domain-transport-key", key.EncryptedSecret, transportSecretAAD(key.DomainID, key.KeyID))
	if err != nil {
		return DomainTransportKey{}, err
	}
	key.Secret = encodeTransportSecret(secret)
	key.EncryptedSecret = ""
	return key, nil
}
