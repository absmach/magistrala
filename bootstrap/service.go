// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"strings"
	"time"

	"github.com/absmach/magistrala"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

var (
	// ErrDeviceBootstrapAuth hides which part of device authentication failed.
	ErrDeviceBootstrapAuth = errors.NewAuthZError("failed to authenticate bootstrap device")

	// ErrBootstrap indicates error in getting bootstrap configuration.
	ErrBootstrap = errors.New("failed to read bootstrap configuration")

	// ErrAddBootstrap indicates error in adding bootstrap configuration.
	ErrAddBootstrap = errors.NewServiceError("failed to add bootstrap configuration")

	// ErrBootstrapStatus indicates an invalid bootstrap status.
	ErrBootstrapStatus = errors.NewRequestError("invalid bootstrap status")

	// ErrBootstrapKey indicates an invalid per-device Bootstrap key.
	ErrBootstrapKey = errors.NewRequestError("invalid bootstrap key")

	// ErrSecretEncryption indicates that encrypted secret storage is unavailable.
	ErrSecretEncryption = errors.NewServiceError("secret encryption is not configured")

	errRemoveBootstrap = errors.New("failed to remove bootstrap configuration")
	errEnableConfig    = errors.New("failed to enable bootstrap configuration")
	errDisableConfig   = errors.New("failed to disable bootstrap configuration")
	errUpdateCert      = errors.New("failed to update cert")

	errCreateProfile           = errors.New("failed to create profile")
	errViewProfile             = errors.New("failed to view profile")
	errUpdateProfile           = errors.New("failed to update profile")
	errDeleteProfile           = errors.New("failed to delete profile")
	errListProfiles            = errors.New("failed to list profiles")
	errAssignProfile           = errors.New("failed to assign profile to enrollment")
	errBindResources           = errors.New("failed to bind resources")
	errListBindings            = errors.New("failed to list bindings")
	errRefreshBinding          = errors.New("failed to refresh bindings")
	errRenderBootstrap         = errors.New("failed to render bootstrap configuration")
	errIssueBootstrapChallenge = errors.New("failed to issue bootstrap challenge")
)

var _ Service = (*bootstrapService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Add adds new Device Config to the user identified by the provided token.
	Add(ctx context.Context, session smqauthn.Session, token string, cfg Config) (Config, error)

	// View returns Device Config with given ID belonging to the user identified by the given token.
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

	// IssueBootstrapChallenge creates a short-lived challenge for an enrollment.
	IssueBootstrapChallenge(ctx context.Context, externalID string) (BootstrapChallengeResponse, error)

	// Bootstrap authenticates a device proof and returns its rendered Config.
	Bootstrap(ctx context.Context, externalID string, proof DeviceBootstrapProof) (Config, error)

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

	// ListProfiles returns a page of Profiles belonging to the workspace.
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
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
type ConfigReader interface {
	ReadConfig(Config) (any, error)
}

type bootstrapService struct {
	configs    ConfigRepository
	profiles   ProfileRepository
	bindings   BindingStore
	challenges BootstrapChallengeRepository
	resolver   BindingResolver
	renderer   Renderer
	dbCipher   *SecretCipher
	idProvider magistrala.IDProvider
	now        func() time.Time
}

// New returns a new Bootstrap service.
//
// An unusable encryption key is reported here rather than deferred: without a
// cipher the service constructs cleanly and then fails every enrollment write
// at runtime.
//
// previousKeys carries retired encryption keys so that secrets sealed before
// a key rotation remain readable; see NewSecretCipher.
//
// challenges may be nil for a deployment that does not serve device
// bootstrap; the challenge and bootstrap endpoints then fail closed.
func New(
	configs ConfigRepository,
	profiles ProfileRepository,
	bindings BindingStore,
	challenges BootstrapChallengeRepository,
	resolver BindingResolver,
	renderer Renderer,
	encKey []byte,
	encKeyID string,
	idp magistrala.IDProvider,
	previousKeys ...PreviousKey,
) (Service, error) {
	dbCipher, err := NewSecretCipher(encKey, encKeyID, previousKeys...)
	if err != nil {
		return nil, err
	}
	return &bootstrapService{
		configs:    configs,
		profiles:   profiles,
		bindings:   bindings,
		challenges: challenges,
		resolver:   resolver,
		renderer:   renderer,
		dbCipher:   dbCipher,
		idProvider: idp,
		now:        time.Now,
	}, nil
}

func (bs bootstrapService) Add(ctx context.Context, session smqauthn.Session, token string, cfg Config) (Config, error) {
	id, err := bs.idProvider.ID()
	if err != nil {
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}

	cfg.ID = id
	cfg.WorkspaceID = session.WorkspaceID
	cfg.Status = Active
	cfg.BootstrapKeyVersion = 1
	// The profile foreign key is workspace-agnostic, so an unvalidated
	// ProfileID would let an enrollment reference another workspace's profile
	// (rendering then fails forever with a 500) or a nonexistent one (which
	// surfaces as a foreign-key violation rather than a bad request). Apply
	// the same check AssignProfile makes.
	if cfg.ProfileID != "" {
		if bs.profiles == nil {
			return Config{}, errors.Wrap(ErrAddBootstrap, errors.New("profile repository not configured"))
		}
		if _, err := bs.profiles.RetrieveByID(ctx, session.WorkspaceID, cfg.ProfileID); err != nil {
			return Config{}, errors.Wrap(svcerr.ErrCreateEntity, err)
		}
	}
	if cfg.ExternalKey == "" {
		cfg.ExternalKey, err = generateBootstrapKey()
		if err != nil {
			return Config{}, errors.Wrap(ErrAddBootstrap, err)
		}
	}
	if _, err := bootstrapKeyMaterial(cfg.ExternalKey); err != nil {
		return Config{}, errors.Wrap(ErrBootstrapKey, err)
	}
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
	cfg, err := bs.configs.RetrieveByID(ctx, session.WorkspaceID, id)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return bs.decryptConfigExternalKeyForManagement(cfg, svcerr.ErrViewEntity)
}

func (bs bootstrapService) Update(ctx context.Context, session smqauthn.Session, cfg Config) error {
	cfg.WorkspaceID = session.WorkspaceID
	if cfg.ExternalKey != "" {
		if _, err := bootstrapKeyMaterial(cfg.ExternalKey); err != nil {
			return errors.Wrap(ErrBootstrapKey, err)
		}
		stored, err := bs.configs.RetrieveByID(ctx, session.WorkspaceID, cfg.ID)
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
	cfg, err := bs.configs.UpdateCert(ctx, session.WorkspaceID, id, clientCert, clientKey, caCert)
	if err != nil {
		return Config{}, errors.Wrap(errUpdateCert, err)
	}
	return cfg, nil
}

func (bs bootstrapService) List(ctx context.Context, session smqauthn.Session, filter Filter, offset, limit uint64) (ConfigsPage, error) {
	page, err := bs.configs.RetrieveAll(ctx, session.WorkspaceID, filter, offset, limit)
	if err != nil {
		return ConfigsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
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
	if err := bs.configs.Remove(ctx, session.WorkspaceID, id); err != nil {
		return errors.Wrap(errRemoveBootstrap, err)
	}
	return nil
}

func (bs bootstrapService) IssueBootstrapChallenge(ctx context.Context, externalID string) (BootstrapChallengeResponse, error) {
	challengeID, err := bs.idProvider.ID()
	if err != nil {
		return BootstrapChallengeResponse{}, errors.Wrap(errIssueBootstrapChallenge, err)
	}
	serverNonce := make([]byte, BootstrapNonceSize)
	if _, err := rand.Read(serverNonce); err != nil {
		return BootstrapChallengeResponse{}, errors.Wrap(errIssueBootstrapChallenge, err)
	}
	now := bs.now().UTC()
	response := BootstrapChallengeResponse{
		ChallengeID: challengeID,
		ServerNonce: encodeBootstrapBytes(serverNonce),
		ExpiresAt:   now.Add(DefaultChallengeTTL),
		KeyVersion:  1,
	}

	cfg, err := bs.configs.RetrieveByExternalID(ctx, externalID)
	if err != nil {
		if !errors.Contains(err, repoerr.ErrNotFound) {
			return BootstrapChallengeResponse{}, errors.Wrap(errIssueBootstrapChallenge, err)
		}
		response.KeyVersion = bs.decoyKeyVersion(externalID)
		return response, nil
	}
	if cfg.Status == DisabledStatus || bs.challenges == nil {
		// Unknown and disabled enrollments receive the same fake challenge
		// response shape. The decoy key version is derived from the external
		// ID under the server's key so it neither is constant (which would
		// mark every non-1 answer as a real, rotated enrollment) nor varies
		// between probes of the same ID.
		response.KeyVersion = bs.decoyKeyVersion(externalID)
		return response, nil
	}
	if cfg.BootstrapKeyVersion == 0 {
		cfg.BootstrapKeyVersion = 1
	}
	response.KeyVersion = cfg.BootstrapKeyVersion
	challenge := BootstrapChallenge{
		ID: challengeID, ConfigID: cfg.ID, KeyVersion: cfg.BootstrapKeyVersion,
		ServerNonce: serverNonce, CreatedAt: now, ExpiresAt: response.ExpiresAt,
	}
	if err := bs.challenges.Create(ctx, challenge); err != nil {
		return BootstrapChallengeResponse{}, errors.Wrap(errIssueBootstrapChallenge, err)
	}
	return response, nil
}

func (bs bootstrapService) Bootstrap(ctx context.Context, externalID string, proof DeviceBootstrapProof) (Config, error) {
	cfg, err := bs.configs.RetrieveByExternalID(ctx, externalID)
	if err != nil {
		if errors.Contains(err, repoerr.ErrNotFound) {
			return Config{}, ErrDeviceBootstrapAuth
		}
		return Config{}, errors.Wrap(ErrBootstrap, err)
	}
	if cfg.Status == DisabledStatus || bs.challenges == nil {
		return Config{}, ErrDeviceBootstrapAuth
	}
	if cfg.BootstrapKeyVersion == 0 {
		cfg.BootstrapKeyVersion = 1
	}
	challenge, err := bs.challenges.Retrieve(ctx, proof.ChallengeID, cfg.ID)
	if err != nil || challenge.ConsumedAt != nil || challenge.KeyVersion != cfg.BootstrapKeyVersion || !bs.now().UTC().Before(challenge.ExpiresAt) {
		return Config{}, ErrDeviceBootstrapAuth
	}
	deviceNonce, err := decodeBootstrapField("device nonce", proof.DeviceNonce, BootstrapNonceSize)
	if err != nil {
		return Config{}, ErrDeviceBootstrapAuth
	}
	providedProof, err := decodeBootstrapField("proof", proof.Proof, BootstrapProofSize)
	if err != nil {
		return Config{}, ErrDeviceBootstrapAuth
	}

	decrypted, err := bs.decryptConfigExternalKey(cfg, ErrDeviceBootstrapAuth)
	if err != nil {
		return Config{}, ErrDeviceBootstrapAuth
	}
	rootKey, err := bootstrapKeyMaterial(decrypted.ExternalKey)
	if err != nil {
		return Config{}, ErrDeviceBootstrapAuth
	}
	serverNonce := encodeBootstrapBytes(challenge.ServerNonce)
	expectedProof, err := calculateBootstrapProof(
		rootKey, cfg.ExternalID, challenge.ID, serverNonce, proof.DeviceNonce, challenge.KeyVersion,
	)
	if err != nil || !hmac.Equal(providedProof, expectedProof) {
		return Config{}, ErrDeviceBootstrapAuth
	}
	now := bs.now().UTC()
	if err := bs.challenges.Consume(ctx, challenge.ID, cfg.ID, now); err != nil {
		return Config{}, ErrDeviceBootstrapAuth
	}

	cfg = decrypted
	cfg, err = bs.renderBootstrapConfig(ctx, cfg)
	if err != nil {
		return Config{}, errors.Wrap(ErrBootstrap, err)
	}
	cfg.ExternalKey = ""
	cfg.BootstrapRootKey = rootKey
	cfg.BootstrapChallengeID = challenge.ID
	cfg.BootstrapServerNonce = serverNonce
	cfg.BootstrapDeviceNonce = encodeBootstrapBytes(deviceNonce)
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
		cfg.ContentType = ContentTypeTextPlain
		return cfg, nil
	}
	if bs.profiles == nil || bs.bindings == nil || bs.renderer == nil {
		return Config{}, errors.Wrap(errRenderBootstrap, errors.New("profile rendering support not configured"))
	}

	profile, err := bs.profiles.RetrieveByID(ctx, cfg.WorkspaceID, cfg.ProfileID)
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
	cfg.ContentType = profile.ContentType
	if cfg.ContentType == "" {
		cfg.ContentType = defaultContentType(profile.ContentFormat)
	}
	return cfg, nil
}

func (bs bootstrapService) EnableConfig(ctx context.Context, session smqauthn.Session, id string) (Config, error) {
	cfg, err := bs.changeConfigStatus(ctx, session.WorkspaceID, id, EnabledStatus)
	if err != nil {
		return Config{}, errors.Wrap(errEnableConfig, err)
	}
	return cfg, nil
}

func (bs bootstrapService) DisableConfig(ctx context.Context, session smqauthn.Session, id string) (Config, error) {
	cfg, err := bs.changeConfigStatus(ctx, session.WorkspaceID, id, DisabledStatus)
	if err != nil {
		return Config{}, errors.Wrap(errDisableConfig, err)
	}
	return cfg, nil
}

func (bs bootstrapService) changeConfigStatus(ctx context.Context, workspaceID, id string, status Status) (Config, error) {
	cfg, err := bs.configs.RetrieveByID(ctx, workspaceID, id)
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
	if err := bs.configs.ChangeStatus(ctx, workspaceID, id, status); err != nil {
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
	p.WorkspaceID = session.WorkspaceID
	if p.ContentFormat == "" {
		p.ContentFormat = ContentFormatJSON
	}
	if p.ContentType == "" {
		p.ContentType = defaultContentType(p.ContentFormat)
	}
	if !validContentType(p.ContentType) {
		return Profile{}, errors.Wrap(errCreateProfile, errors.NewRequestError("unsupported profile content type"))
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
	p, err := bs.profiles.RetrieveByID(ctx, session.WorkspaceID, profileID)
	if err != nil {
		return Profile{}, errors.Wrap(errViewProfile, err)
	}
	return p, nil
}

func (bs bootstrapService) UpdateProfile(ctx context.Context, session smqauthn.Session, p Profile) (Profile, error) {
	if bs.profiles == nil {
		return Profile{}, errors.Wrap(errUpdateProfile, errors.New("profile repository not configured"))
	}
	p.WorkspaceID = session.WorkspaceID
	if p.ContentType != "" && !validContentType(p.ContentType) {
		return Profile{}, errors.Wrap(errUpdateProfile, errors.NewRequestError("unsupported profile content type"))
	}
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
	page, err := bs.profiles.RetrieveAll(ctx, session.WorkspaceID, offset, limit, name)
	if err != nil {
		return ProfilesPage{}, errors.Wrap(errListProfiles, err)
	}
	return page, nil
}

func (bs bootstrapService) DeleteProfile(ctx context.Context, session smqauthn.Session, profileID string) error {
	if bs.profiles == nil {
		return errors.Wrap(errDeleteProfile, errors.New("profile repository not configured"))
	}
	if err := bs.profiles.Delete(ctx, session.WorkspaceID, profileID); err != nil {
		return errors.Wrap(errDeleteProfile, err)
	}
	return nil
}

// --- Enrollment-profile assignment ---

func (bs bootstrapService) AssignProfile(ctx context.Context, session smqauthn.Session, configID, profileID string) error {
	if bs.profiles == nil {
		return errors.Wrap(errAssignProfile, errors.New("profile repository not configured"))
	}
	// Validate profile exists in workspace.
	if _, err := bs.profiles.RetrieveByID(ctx, session.WorkspaceID, profileID); err != nil {
		return errors.Wrap(errAssignProfile, err)
	}
	if err := bs.configs.AssignProfile(ctx, session.WorkspaceID, configID, profileID); err != nil {
		return errors.Wrap(errAssignProfile, err)
	}
	return nil
}

// --- Binding management ---

func (bs bootstrapService) BindResources(ctx context.Context, session smqauthn.Session, token, configID string, requested []BindingRequest) error {
	if bs.profiles == nil || bs.bindings == nil || bs.resolver == nil {
		return errors.Wrap(errBindResources, errors.New("binding support not configured"))
	}
	cfg, err := bs.configs.RetrieveByID(ctx, session.WorkspaceID, configID)
	if err != nil {
		return errors.Wrap(errBindResources, err)
	}
	profile, err := bs.profiles.RetrieveByID(ctx, session.WorkspaceID, cfg.ProfileID)
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
	if _, err := bs.configs.RetrieveByID(ctx, session.WorkspaceID, configID); err != nil {
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
	cfg, err := bs.configs.RetrieveByID(ctx, session.WorkspaceID, configID)
	if err != nil {
		return errors.Wrap(errRefreshBinding, err)
	}
	profile, err := bs.profiles.RetrieveByID(ctx, session.WorkspaceID, cfg.ProfileID)
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
