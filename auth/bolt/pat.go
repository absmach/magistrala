// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bolt

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	bolt "go.etcd.io/bbolt"
)

const (
	idKey               = "id"
	userKey             = "user"
	nameKey             = "name"
	descriptionKey      = "description"
	secretKey           = "secret_key"
	scopeKey            = "scope"
	issuedAtKey         = "issued_at"
	expiresAtKey        = "expires_at"
	updatedAtKey        = "updated_at"
	lastUsedAtKey       = "last_used_at"
	revokedKey          = "revoked"
	revokedAtKey        = "revoked_at"
	platformEntitiesKey = "platform_entities"
	patKey              = "pat"

	keySeparator = ":"
	anyID        = "*"
)

var (
	activateValue    = []byte{0x00}
	revokedValue     = []byte{0x01}
	entityValue      = []byte{0x02}
	anyIDValue       = []byte{0x03}
	selectedIDsValue = []byte{0x04}
)

type patRepo struct {
	db         *bolt.DB
	bucketName string
}

// NewPATSRepository instantiates a bolt
// implementation of PAT repository.
func NewPATSRepository(db *bolt.DB, bucketName string) auth.PATSRepository {
	return &patRepo{
		db:         db,
		bucketName: bucketName,
	}
}

func (pr *patRepo) Save(ctx context.Context, pat auth.PAT) error {
	idxKey := []byte(pat.User + keySeparator + patKey + keySeparator + pat.ID)
	kv, err := patToKeyValue(pat)
	if err != nil {
		return err
	}
	return pr.db.Update(func(tx *bolt.Tx) error {
		rootBucket, err := pr.retrieveRootBucket(tx)
		if err != nil {
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		b, err := pr.createUserBucket(rootBucket, pat.User)
		if err != nil {
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		for key, value := range kv {
			fullKey := []byte(pat.ID + keySeparator + key)
			if err := b.Put(fullKey, value); err != nil {
				return errors.Wrap(repoerr.ErrCreateEntity, err)
			}
		}
		if err := rootBucket.Put(idxKey, []byte(pat.ID)); err != nil {
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		return nil
	})
}

func (pr *patRepo) Retrieve(ctx context.Context, userID, patID string) (auth.PAT, error) {
	prefix := []byte(patID + keySeparator)
	kv := map[string][]byte{}
	if err := pr.db.View(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrViewEntity)
		if err != nil {
			return err
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			kv[string(k)] = v
		}
		return nil
	}); err != nil {
		return auth.PAT{}, err
	}

	return keyValueToPAT(kv)
}

func (pr *patRepo) RetrieveSecretAndRevokeStatus(ctx context.Context, userID, patID string) (string, bool, error) {
	revoked := true
	keySecret := patID + keySeparator + secretKey
	keyRevoked := patID + keySeparator + revokedKey
	var secretHash string
	if err := pr.db.View(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrViewEntity)
		if err != nil {
			return err
		}
		secretHash = string(b.Get([]byte(keySecret)))
		revoked = bytesToBoolean(b.Get([]byte(keyRevoked)))
		return nil
	}); err != nil {
		return "", true, err
	}
	return secretHash, revoked, nil
}

func (pr *patRepo) UpdateName(ctx context.Context, userID, patID, name string) (auth.PAT, error) {
	return pr.updatePATField(ctx, userID, patID, nameKey, []byte(name))
}

func (pr *patRepo) UpdateDescription(ctx context.Context, userID, patID, description string) (auth.PAT, error) {
	return pr.updatePATField(ctx, userID, patID, descriptionKey, []byte(description))
}

func (pr *patRepo) UpdateTokenHash(ctx context.Context, userID, patID, tokenHash string, expiryAt time.Time) (auth.PAT, error) {
	prefix := []byte(patID + keySeparator)
	kv := map[string][]byte{}
	if err := pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrUpdateEntity)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(patID+keySeparator+secretKey), []byte(tokenHash)); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		if err := b.Put([]byte(patID+keySeparator+expiresAtKey), timeToBytes(expiryAt)); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		if err := b.Put([]byte(patID+keySeparator+updatedAtKey), timeToBytes(time.Now())); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			kv[string(k)] = v
		}
		return nil
	}); err != nil {
		return auth.PAT{}, err
	}
	return keyValueToPAT(kv)
}

func (pr *patRepo) RetrieveAll(ctx context.Context, userID string, pm auth.PATSPageMeta) (auth.PATSPage, error) {
	prefix := []byte(userID + keySeparator + patKey + keySeparator)

	patIDs := []string{}
	if err := pr.db.View(func(tx *bolt.Tx) error {
		b, err := pr.retrieveRootBucket(tx)
		if err != nil {
			return errors.Wrap(repoerr.ErrViewEntity, err)
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if v != nil {
				patIDs = append(patIDs, string(v))
			}
		}
		return nil
	}); err != nil {
		return auth.PATSPage{}, err
	}

	total := len(patIDs)

	var pats []auth.PAT

	patsPage := auth.PATSPage{
		Total:  uint64(total),
		Limit:  pm.Limit,
		Offset: pm.Offset,
		PATS:   pats,
	}

	if int(pm.Offset) >= total {
		return patsPage, nil
	}

	aLimit := pm.Limit
	if rLimit := total - int(pm.Offset); int(pm.Limit) > rLimit {
		aLimit = uint64(rLimit)
	}

	for i := pm.Offset; i < pm.Offset+aLimit; i++ {
		if int(i) < total {
			pat, err := pr.Retrieve(ctx, userID, patIDs[i])
			if err != nil {
				return patsPage, err
			}
			patsPage.PATS = append(patsPage.PATS, pat)
		}
	}

	return patsPage, nil
}

func (pr *patRepo) Revoke(ctx context.Context, userID, patID string) error {
	if err := pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrUpdateEntity)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(patID+keySeparator+revokedKey), revokedValue); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		if err := b.Put([]byte(patID+keySeparator+revokedAtKey), timeToBytes(time.Now())); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (pr *patRepo) Reactivate(ctx context.Context, userID, patID string) error {
	if err := pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrUpdateEntity)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(patID+keySeparator+revokedKey), activateValue); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		if err := b.Put([]byte(patID+keySeparator+revokedAtKey), []byte{}); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (pr *patRepo) Remove(ctx context.Context, userID, patID string) error {
	prefix := []byte(patID + keySeparator)
	idxKey := []byte(userID + keySeparator + patKey + keySeparator + patID)
	if err := pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrRemoveEntity)
		if err != nil {
			return err
		}
		c := b.Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			if err := b.Delete(k); err != nil {
				return errors.Wrap(repoerr.ErrRemoveEntity, err)
			}
		}
		rb, err := pr.retrieveRootBucket(tx)
		if err != nil {
			return err
		}
		if err := rb.Delete(idxKey); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (pr *patRepo) AddScopeEntry(ctx context.Context, userID, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) (auth.Scope, error) {
	prefix := []byte(patID + keySeparator + scopeKey)
	var rKV map[string][]byte
	if err := pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrCreateEntity)
		if err != nil {
			return err
		}
		kv, err := scopeEntryToKeyValue(platformEntityType, optionalDomainID, optionalDomainEntityType, operation, entityIDs...)
		if err != nil {
			return err
		}
		for key, value := range kv {
			fullKey := []byte(patID + keySeparator + key)
			if err := b.Put(fullKey, value); err != nil {
				return errors.Wrap(repoerr.ErrCreateEntity, err)
			}
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			rKV[string(k)] = v
		}
		return nil
	}); err != nil {
		return auth.Scope{}, err
	}

	return parseKeyValueToScope(rKV)
}

func (pr *patRepo) RemoveScopeEntry(ctx context.Context, userID, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) (auth.Scope, error) {
	if len(entityIDs) == 0 {
		return auth.Scope{}, repoerr.ErrMalformedEntity
	}
	prefix := []byte(patID + keySeparator + scopeKey)
	var rKV map[string][]byte
	if err := pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrRemoveEntity)
		if err != nil {
			return err
		}
		kv, err := scopeEntryToKeyValue(platformEntityType, optionalDomainID, optionalDomainEntityType, operation, entityIDs...)
		if err != nil {
			return err
		}
		for key := range kv {
			fullKey := []byte(patID + keySeparator + key)
			if err := b.Delete(fullKey); err != nil {
				return errors.Wrap(repoerr.ErrRemoveEntity, err)
			}
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			rKV[string(k)] = v
		}
		return nil
	}); err != nil {
		return auth.Scope{}, err
	}
	return parseKeyValueToScope(rKV)
}

func (pr *patRepo) CheckScopeEntry(ctx context.Context, userID, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) error {
	return pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrViewEntity)
		if err != nil {
			return errors.Wrap(repoerr.ErrViewEntity, err)
		}
		srootKey, err := scopeRootKey(platformEntityType, optionalDomainID, optionalDomainEntityType, operation)
		if err != nil {
			return errors.Wrap(repoerr.ErrViewEntity, err)
		}

		rootKey := patID + keySeparator + srootKey
		if value := b.Get([]byte(rootKey)); bytes.Equal(value, anyIDValue) {
			return nil
		}
		for _, entity := range entityIDs {
			value := b.Get([]byte(rootKey + keySeparator + entity))
			if !bytes.Equal(value, entityValue) {
				return repoerr.ErrNotFound
			}
		}
		return nil
	})
}

func (pr *patRepo) RemoveAllScopeEntry(ctx context.Context, userID, patID string) error {
	return nil
}

func (pr *patRepo) updatePATField(_ context.Context, userID, patID, key string, value []byte) (auth.PAT, error) {
	prefix := []byte(patID + keySeparator)
	kv := map[string][]byte{}
	if err := pr.db.Update(func(tx *bolt.Tx) error {
		b, err := pr.retrieveUserBucket(tx, userID, patID, repoerr.ErrUpdateEntity)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(patID+keySeparator+key), value); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		if err := b.Put([]byte(patID+keySeparator+updatedAtKey), timeToBytes(time.Now())); err != nil {
			return errors.Wrap(repoerr.ErrUpdateEntity, err)
		}
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			kv[string(k)] = v
		}
		return nil
	}); err != nil {
		return auth.PAT{}, err
	}
	return keyValueToPAT(kv)
}

func (pr *patRepo) createUserBucket(rootBucket *bolt.Bucket, userID string) (*bolt.Bucket, error) {
	userBucket, err := rootBucket.CreateBucketIfNotExists([]byte(userID))
	if err != nil {
		return nil, errors.Wrap(repoerr.ErrCreateEntity, fmt.Errorf("failed to retrieve or create bucket for user %s : %w", userID, err))
	}

	return userBucket, nil
}

func (pr *patRepo) retrieveUserBucket(tx *bolt.Tx, userID, patID string, wrap error) (*bolt.Bucket, error) {
	rootBucket, err := pr.retrieveRootBucket(tx)
	if err != nil {
		return nil, errors.Wrap(wrap, err)
	}

	vPatID := rootBucket.Get([]byte(userID + keySeparator + patKey + keySeparator + patID))
	if vPatID == nil {
		return nil, repoerr.ErrNotFound
	}

	userBucket := rootBucket.Bucket([]byte(userID))
	if userBucket == nil {
		return nil, errors.Wrap(wrap, fmt.Errorf("user %s not found", userID))
	}
	return userBucket, nil
}

func (pr *patRepo) retrieveRootBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	rootBucket := tx.Bucket([]byte(pr.bucketName))
	if rootBucket == nil {
		return nil, fmt.Errorf("bucket %s not found", pr.bucketName)
	}
	return rootBucket, nil
}

func patToKeyValue(pat auth.PAT) (map[string][]byte, error) {
	kv := map[string][]byte{
		idKey:          []byte(pat.ID),
		userKey:        []byte(pat.User),
		nameKey:        []byte(pat.Name),
		descriptionKey: []byte(pat.Description),
		secretKey:      []byte(pat.Secret),
		issuedAtKey:    timeToBytes(pat.IssuedAt),
		expiresAtKey:   timeToBytes(pat.ExpiresAt),
		updatedAtKey:   timeToBytes(pat.UpdatedAt),
		lastUsedAtKey:  timeToBytes(pat.LastUsedAt),
		revokedKey:     booleanToBytes(pat.Revoked),
		revokedAtKey:   timeToBytes(pat.RevokedAt),
	}
	scopeKV, err := scopeToKeyValue(pat.Scope)
	if err != nil {
		return nil, err
	}
	for k, v := range scopeKV {
		kv[k] = v
	}
	return kv, nil
}

func scopeToKeyValue(scope auth.Scope) (map[string][]byte, error) {
	kv := map[string][]byte{}
	for opType, scopeValue := range scope.Users {
		tempKV, err := scopeEntryToKeyValue(auth.PlatformUsersScope, "", auth.DomainNullScope, opType, scopeValue.Values()...)
		if err != nil {
			return nil, err
		}
		for k, v := range tempKV {
			kv[k] = v
		}
	}
	for domainID, domainScope := range scope.Domains {
		for opType, scopeValue := range domainScope.DomainManagement {
			tempKV, err := scopeEntryToKeyValue(auth.PlatformDomainsScope, domainID, auth.DomainManagementScope, opType, scopeValue.Values()...)
			if err != nil {
				return nil, errors.Wrap(repoerr.ErrCreateEntity, err)
			}
			for k, v := range tempKV {
				kv[k] = v
			}
		}
		for entityType, scope := range domainScope.Entities {
			for opType, scopeValue := range scope {
				tempKV, err := scopeEntryToKeyValue(auth.PlatformDomainsScope, domainID, entityType, opType, scopeValue.Values()...)
				if err != nil {
					return nil, errors.Wrap(repoerr.ErrCreateEntity, err)
				}
				for k, v := range tempKV {
					kv[k] = v
				}
			}
		}
	}
	return kv, nil
}

func scopeEntryToKeyValue(platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) (map[string][]byte, error) {
	if len(entityIDs) == 0 {
		return nil, repoerr.ErrMalformedEntity
	}

	rootKey, err := scopeRootKey(platformEntityType, optionalDomainID, optionalDomainEntityType, operation)
	if err != nil {
		return nil, err
	}
	if len(entityIDs) == 1 && entityIDs[0] == anyID {
		return map[string][]byte{rootKey: anyIDValue}, nil
	}

	kv := map[string][]byte{rootKey: selectedIDsValue}

	for _, entryID := range entityIDs {
		if entryID == anyID {
			return nil, repoerr.ErrMalformedEntity
		}
		kv[rootKey+keySeparator+entryID] = entityValue
	}

	return kv, nil
}

func scopeRootKey(platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType) (string, error) {
	op, err := operation.ValidString()
	if err != nil {
		return "", errors.Wrap(repoerr.ErrMalformedEntity, err)
	}

	var rootKey strings.Builder

	rootKey.WriteString(scopeKey)
	rootKey.WriteString(keySeparator)
	rootKey.WriteString(platformEntityType.String())
	rootKey.WriteString(keySeparator)

	switch platformEntityType {
	case auth.PlatformUsersScope:
		rootKey.WriteString(op)
	case auth.PlatformDomainsScope:
		if optionalDomainID == "" {
			return "", fmt.Errorf("failed to add platform %s scope: invalid domain id", platformEntityType.String())
		}
		odet, err := optionalDomainEntityType.ValidString()
		if err != nil {
			return "", errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		rootKey.WriteString(optionalDomainID)
		rootKey.WriteString(keySeparator)
		rootKey.WriteString(odet)
		rootKey.WriteString(keySeparator)
		rootKey.WriteString(op)
	default:
		return "", errors.Wrap(repoerr.ErrMalformedEntity, fmt.Errorf("invalid platform entity type %s", platformEntityType.String()))
	}

	return rootKey.String(), nil
}

func keyValueToBasicPAT(kv map[string][]byte) auth.PAT {
	var pat auth.PAT
	for k, v := range kv {
		switch {
		case strings.HasSuffix(k, keySeparator+idKey):
			pat.ID = string(v)
		case strings.HasSuffix(k, keySeparator+userKey):
			pat.User = string(v)
		case strings.HasSuffix(k, keySeparator+nameKey):
			pat.Name = string(v)
		case strings.HasSuffix(k, keySeparator+descriptionKey):
			pat.Description = string(v)
		case strings.HasSuffix(k, keySeparator+issuedAtKey):
			pat.IssuedAt = bytesToTime(v)
		case strings.HasSuffix(k, keySeparator+expiresAtKey):
			pat.ExpiresAt = bytesToTime(v)
		case strings.HasSuffix(k, keySeparator+updatedAtKey):
			pat.UpdatedAt = bytesToTime(v)
		case strings.HasSuffix(k, keySeparator+lastUsedAtKey):
			pat.LastUsedAt = bytesToTime(v)
		case strings.HasSuffix(k, keySeparator+revokedKey):
			pat.Revoked = bytesToBoolean(v)
		case strings.HasSuffix(k, keySeparator+revokedAtKey):
			pat.RevokedAt = bytesToTime(v)
		}
	}
	return pat
}

func keyValueToPAT(kv map[string][]byte) (auth.PAT, error) {
	pat := keyValueToBasicPAT(kv)
	scope, err := parseKeyValueToScope(kv)
	if err != nil {
		return auth.PAT{}, err
	}
	pat.Scope = scope
	return pat, nil
}

func parseKeyValueToScope(kv map[string][]byte) (auth.Scope, error) {
	scope := auth.Scope{
		Domains: make(map[string]auth.DomainScope),
	}
	for key, value := range kv {
		if strings.Index(key, keySeparator+scopeKey+keySeparator) > 0 {
			keyParts := strings.Split(key, keySeparator)

			platformEntityType, err := auth.ParsePlatformEntityType(keyParts[2])
			if err != nil {
				return auth.Scope{}, errors.Wrap(repoerr.ErrViewEntity, err)
			}

			switch platformEntityType {
			case auth.PlatformUsersScope:
				scope.Users, err = parseOperation(platformEntityType, scope.Users, key, keyParts, value)
				if err != nil {
					return auth.Scope{}, errors.Wrap(repoerr.ErrViewEntity, err)
				}

			case auth.PlatformDomainsScope:
				if len(keyParts) < 6 {
					return auth.Scope{}, fmt.Errorf("invalid scope key format: %s", key)
				}
				domainID := keyParts[3]
				if scope.Domains == nil {
					scope.Domains = make(map[string]auth.DomainScope)
				}
				if _, ok := scope.Domains[domainID]; !ok {
					scope.Domains[domainID] = auth.DomainScope{}
				}
				domainScope := scope.Domains[domainID]

				entityType := keyParts[4]

				switch entityType {
				case auth.DomainManagementScope.String():
					domainScope.DomainManagement, err = parseOperation(platformEntityType, domainScope.DomainManagement, key, keyParts, value)
					if err != nil {
						return auth.Scope{}, errors.Wrap(repoerr.ErrViewEntity, err)
					}
				default:
					etype, err := auth.ParseDomainEntityType(entityType)
					if err != nil {
						return auth.Scope{}, fmt.Errorf("key %s invalid entity type %s : %w", key, entityType, err)
					}
					if domainScope.Entities == nil {
						domainScope.Entities = make(map[auth.DomainEntityType]auth.OperationScope)
					}
					if _, ok := domainScope.Entities[etype]; !ok {
						domainScope.Entities[etype] = auth.OperationScope{}
					}
					entityOperationScope := domainScope.Entities[etype]
					entityOperationScope, err = parseOperation(platformEntityType, entityOperationScope, key, keyParts, value)
					if err != nil {
						return auth.Scope{}, errors.Wrap(repoerr.ErrViewEntity, err)
					}
					domainScope.Entities[etype] = entityOperationScope
				}
				scope.Domains[domainID] = domainScope
			default:
				return auth.Scope{}, errors.Wrap(repoerr.ErrViewEntity, fmt.Errorf("invalid platform entity type : %s", platformEntityType.String()))
			}
		}
	}
	return scope, nil
}

func parseOperation(platformEntityType auth.PlatformEntityType, opScope auth.OperationScope, key string, keyParts []string, value []byte) (auth.OperationScope, error) {
	if opScope == nil {
		opScope = make(map[auth.OperationType]auth.ScopeValue)
	}

	if err := validateOperation(platformEntityType, opScope, key, keyParts, value); err != nil {
		return auth.OperationScope{}, err
	}

	switch string(value) {
	case string(entityValue):
		opType, err := auth.ParseOperationType(keyParts[len(keyParts)-2])
		if err != nil {
			return auth.OperationScope{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		entityID := keyParts[len(keyParts)-1]

		if _, oValueExists := opScope[opType]; !oValueExists {
			opScope[opType] = &auth.SelectedIDs{}
		}
		oValue := opScope[opType]
		if err := oValue.AddValues(entityID); err != nil {
			return auth.OperationScope{}, fmt.Errorf("failed to add scope key %s with entity value %v : %w", key, entityID, err)
		}
		opScope[opType] = oValue
	case string(anyIDValue):
		opType, err := auth.ParseOperationType(keyParts[len(keyParts)-1])
		if err != nil {
			return auth.OperationScope{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		if oValue, oValueExists := opScope[opType]; oValueExists && oValue != nil {
			if _, ok := oValue.(*auth.AnyIDs); !ok {
				return auth.OperationScope{}, fmt.Errorf("failed to add scope key %s with entity anyIDs scope value : key already initialized with different type", key)
			}
		}
		opScope[opType] = &auth.AnyIDs{}
	case string(selectedIDsValue):
		opType, err := auth.ParseOperationType(keyParts[len(keyParts)-1])
		if err != nil {
			return auth.OperationScope{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}
		oValue, oValueExists := opScope[opType]
		if oValueExists && oValue != nil {
			if _, ok := oValue.(*auth.SelectedIDs); !ok {
				return auth.OperationScope{}, fmt.Errorf("failed to add scope key %s with entity selectedIDs scope value : key already initialized with different type", key)
			}
		}
		if !oValueExists {
			opScope[opType] = &auth.SelectedIDs{}
		}
	default:
		return auth.OperationScope{}, fmt.Errorf("key %s have invalid value %v", key, value)
	}
	return opScope, nil
}

func validateOperation(platformEntityType auth.PlatformEntityType, opScope auth.OperationScope, key string, keyParts []string, value []byte) error {
	expectedKeyPartsLength := 0
	switch string(value) {
	case string(entityValue):
		switch platformEntityType {
		case auth.PlatformDomainsScope:
			expectedKeyPartsLength = 7
		case auth.PlatformUsersScope:
			expectedKeyPartsLength = 5
		default:
			return fmt.Errorf("invalid platform entity type : %s", platformEntityType.String())
		}
	case string(selectedIDsValue), string(anyIDValue):
		switch platformEntityType {
		case auth.PlatformDomainsScope:
			expectedKeyPartsLength = 6
		case auth.PlatformUsersScope:
			expectedKeyPartsLength = 4
		default:
			return fmt.Errorf("invalid platform entity type : %s", platformEntityType.String())
		}
	default:
		return fmt.Errorf("key %s have invalid value %v", key, value)
	}
	if len(keyParts) != expectedKeyPartsLength {
		return fmt.Errorf("invalid scope key format: %s", key)
	}
	return nil
}

func timeToBytes(t time.Time) []byte {
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(t.Unix()))
	return timeBytes
}

func bytesToTime(b []byte) time.Time {
	timeAtSeconds := binary.BigEndian.Uint64(b)
	return time.Unix(int64(timeAtSeconds), 0)
}

func booleanToBytes(b bool) []byte {
	if b {
		return []byte{1}
	}
	return []byte{0}
}

func bytesToBoolean(b []byte) bool {
	if len(b) > 1 || b[0] != activateValue[0] {
		return true
	}
	return false
}
