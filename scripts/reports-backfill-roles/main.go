// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main backfills missing built-in roles for reports.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/policies/spicedb"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/roles"
	spicedbdecoder "github.com/absmach/magistrala/pkg/spicedb"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/operations"
	repg "github.com/absmach/magistrala/reports/postgres"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	cmdName = "reports_backfill_roles"
)

var (
	logLevel            = "info"
	dryRun              = false
	limit               = 0
	defaultMemberID     = ""
	spicedbHost         = "localhost"
	spicedbPort         = "50051"
	spicedbPreSharedKey = "12345678"
	spicedbSchemaFile   = "docker/spicedb/combined-schema.zed"
	dbConfig            = pgclient.Config{
		Host:    "localhost",
		Port:    "15432",
		User:    "postgres",
		Pass:    "magistrala",
		Name:    "reports",
		SSLMode: "disable",
	}
)

type missingReport struct {
	ID        string         `db:"id"`
	Name      string         `db:"name"`
	DomainID  string         `db:"domain_id"`
	CreatedBy sql.NullString `db:"created_by"`
}

func main() {
	ctx := context.Background()

	if limit < 0 {
		log.Fatalf("invalid limit %d: limit must be >= 0", limit)
	}

	logger, err := mglog.New(os.Stdout, logLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

	sqlDB, err := pgclient.Connect(dbConfig)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		exitCode = 1
		return
	}
	defer sqlDB.Close()

	database := pgclient.NewDatabase(sqlDB, dbConfig, noop.NewTracerProvider().Tracer(cmdName))
	reportsRepo := repg.NewRepository(database)

	reportsWithoutRoles, err := listReportsWithoutRoles(ctx, database, limit)
	if err != nil {
		logger.Error("failed to list reports without roles", "error", err)
		exitCode = 1
		return
	}

	logger.Info("loaded reports without roles", "count", len(reportsWithoutRoles), "dry_run", dryRun)
	if len(reportsWithoutRoles) == 0 {
		return
	}

	availableActions, builtInRoles, err := availableActionsAndBuiltInRoles(spicedbSchemaFile)
	if err != nil {
		logger.Error("failed to load built-in role actions", "error", err)
		exitCode = 1
		return
	}

	adminRoleActions, err := builtInRoleActionStrings(builtInRoles, reports.BuiltInRoleAdmin)
	if err != nil {
		logger.Error("failed to resolve built-in admin role actions", "error", err)
		exitCode = 1
		return
	}

	authzedClient, err := newAuthzedClient(spicedbHost, spicedbPort, spicedbPreSharedKey)
	if err != nil {
		logger.Error("failed to connect to spicedb", "error", err)
		exitCode = 1
		return
	}

	if dryRun {
		var processed, skipped int

		for _, report := range reportsWithoutRoles {
			memberID := strings.TrimSpace(report.CreatedBy.String)
			if memberID == "" {
				memberID = strings.TrimSpace(defaultMemberID)
			}
			if report.DomainID == "" {
				skipped++
				logger.Warn("skipping report without domain_id", "report_id", report.ID, "name", report.Name)
				continue
			}
			if memberID == "" {
				skipped++
				logger.Warn("skipping report without created_by and no default member override", "report_id", report.ID, "name", report.Name)
				continue
			}

			isDomainMember, err := isDomainRoleMember(ctx, database, report.DomainID, memberID)
			if err != nil {
				skipped++
				logger.Warn(
					"skipping report after failed domain membership check",
					"report_id", report.ID,
					"name", report.Name,
					"domain_id", report.DomainID,
					"member_id", memberID,
					"error", err,
				)
				continue
			}

			candidatePolicies := []policies.Policy{
				{
					SubjectType: policies.DomainType,
					Subject:     report.DomainID,
					Relation:    policies.DomainRelation,
					ObjectType:  operations.EntityType,
					Object:      report.ID,
				},
			}
			policiesToAdd, existingPolicies, err := filterMissingPolicies(ctx, authzedClient.PermissionsServiceClient, candidatePolicies)
			if err != nil {
				skipped++
				logger.Warn(
					"skipping report after failed spicedb policy lookup",
					"report_id", report.ID,
					"name", report.Name,
					"domain_id", report.DomainID,
					"error", err,
				)
				continue
			}
			for _, existing := range existingPolicies {
				logger.Info(
					"dry run: spicedb policy already exists, will not be re-added",
					"report_id", report.ID,
					"subject_type", existing.SubjectType,
					"subject", existing.Subject,
					"relation", existing.Relation,
					"object_type", existing.ObjectType,
					"object", existing.Object,
				)
			}

			if !isDomainMember {
				logger.Warn(
					"created_by user is not a member of the domain; role will be provisioned without member",
					"report_id", report.ID,
					"name", report.Name,
					"domain_id", report.DomainID,
					"member_id", memberID,
					"created_by_exists_in_domain", false,
					"role_actions", adminRoleActions,
				)
				processed++
				logger.Info(
					"dry run: would provision missing built-in role without member",
					"report_id", report.ID,
					"name", report.Name,
					"domain_id", report.DomainID,
					"member_id", memberID,
					"created_by_exists_in_domain", false,
					"role_actions", adminRoleActions,
					"role_name", reports.BuiltInRoleAdmin.String(),
					"new_optional_policies", len(policiesToAdd),
					"existing_optional_policies", len(existingPolicies),
				)
				continue
			}

			processed++
			logger.Info(
				"dry run: would provision missing built-in role",
				"report_id", report.ID,
				"name", report.Name,
				"domain_id", report.DomainID,
				"member_id", memberID,
				"created_by_exists_in_domain", true,
				"role_actions", adminRoleActions,
				"role_name", reports.BuiltInRoleAdmin.String(),
				"new_optional_policies", len(policiesToAdd),
				"existing_optional_policies", len(existingPolicies),
			)
		}

		logger.Info(
			"backfill finished",
			"processed", processed,
			"skipped", skipped,
			"failed", 0,
			"dry_run", true,
		)
		return
	}

	policyService := spicedb.NewPolicyService(authzedClient, logger)

	provisioner, err := roles.NewProvisionManageService(
		operations.EntityType,
		reportsRepo,
		policyService,
		uuid.New(),
		availableActions,
		builtInRoles,
	)
	if err != nil {
		logger.Error("failed to create roles provisioner", "error", err)
		exitCode = 1
		return
	}

	var processed, skipped, failed int

	for _, report := range reportsWithoutRoles {
		memberID := strings.TrimSpace(report.CreatedBy.String)
		if memberID == "" {
			memberID = strings.TrimSpace(defaultMemberID)
		}
		if report.DomainID == "" {
			skipped++
			logger.Warn("skipping report without domain_id", "report_id", report.ID, "name", report.Name)
			continue
		}
		if memberID == "" {
			skipped++
			logger.Warn("skipping report without created_by and no default member override", "report_id", report.ID, "name", report.Name)
			continue
		}

		assignMembers := []roles.Member{}
		isDomainMember, err := isDomainRoleMember(ctx, database, report.DomainID, memberID)
		if err != nil {
			failed++
			logger.Error(
				"failed to check domain membership before provisioning role",
				"report_id", report.ID,
				"name", report.Name,
				"domain_id", report.DomainID,
				"member_id", memberID,
				"error", err,
			)
			continue
		}
		if isDomainMember {
			assignMembers = []roles.Member{roles.Member(memberID)}
		} else {
			logger.Warn(
				"created_by user is not a member of the domain; provisioning role without member",
				"report_id", report.ID,
				"name", report.Name,
				"domain_id", report.DomainID,
				"member_id", memberID,
				"created_by_exists_in_domain", false,
				"role_actions", adminRoleActions,
			)
		}

		candidatePolicies := []policies.Policy{
			{
				SubjectType: policies.DomainType,
				Subject:     report.DomainID,
				Relation:    policies.DomainRelation,
				ObjectType:  operations.EntityType,
				Object:      report.ID,
			},
		}
		optionalPolicies, existingPolicies, err := filterMissingPolicies(ctx, authzedClient.PermissionsServiceClient, candidatePolicies)
		if err != nil {
			failed++
			logger.Error(
				"failed to check existing spicedb policies",
				"report_id", report.ID,
				"name", report.Name,
				"domain_id", report.DomainID,
				"error", err,
			)
			continue
		}
		for _, existing := range existingPolicies {
			logger.Info(
				"spicedb policy already exists, skipping re-add",
				"report_id", report.ID,
				"subject_type", existing.SubjectType,
				"subject", existing.Subject,
				"relation", existing.Relation,
				"object_type", existing.ObjectType,
				"object", existing.Object,
			)
		}

		newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
			reports.BuiltInRoleAdmin: assignMembers,
		}

		if _, err := provisioner.AddNewEntitiesRoles(
			ctx,
			report.DomainID,
			memberID,
			[]string{report.ID},
			optionalPolicies,
			newBuiltInRoleMembers,
		); err != nil {
			failed++
			logger.Error(
				"failed to provision missing built-in role",
				"report_id", report.ID,
				"name", report.Name,
				"domain_id", report.DomainID,
				"member_id", memberID,
				"error", err,
			)
			continue
		}

		processed++
		logger.Info(
			"provisioned missing built-in role",
			"report_id", report.ID,
			"name", report.Name,
			"domain_id", report.DomainID,
			"member_id", memberID,
			"created_by_exists_in_domain", isDomainMember,
			"member_added", len(assignMembers) > 0,
			"role_actions", adminRoleActions,
			"role_name", reports.BuiltInRoleAdmin.String(),
			"new_optional_policies", len(optionalPolicies),
			"existing_optional_policies", len(existingPolicies),
		)
	}

	logger.Info(
		"backfill finished",
		"processed", processed,
		"skipped", skipped,
		"failed", failed,
		"dry_run", dryRun,
	)

	if failed > 0 {
		exitCode = 1
	}
}

func listReportsWithoutRoles(ctx context.Context, db pgclient.Database, limit int) ([]missingReport, error) {
	params := map[string]any{}

	query := `
		SELECT rc.id, rc.name, rc.domain_id, rc.created_by
		FROM report_config rc
		WHERE NOT EXISTS (
			SELECT 1
			FROM reports_roles rr
			WHERE rr.entity_id = rc.id
		)
	`

	query += " ORDER BY rc.created_at ASC NULLS LAST, rc.id ASC"

	if limit > 0 {
		query += " LIMIT :limit"
		params["limit"] = limit
	}

	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return nil, errors.Wrap(fmt.Errorf("failed to query reports without roles"), err)
	}
	defer rows.Close()

	var reps []missingReport
	for rows.Next() {
		var rep missingReport
		if err := rows.StructScan(&rep); err != nil {
			return nil, errors.Wrap(fmt.Errorf("failed to scan report without role"), err)
		}
		reps = append(reps, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(fmt.Errorf("failed to iterate reports without roles"), err)
	}

	return reps, nil
}

func isDomainRoleMember(ctx context.Context, db pgclient.Database, domainID, memberID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM domains_role_members drm
			WHERE drm.entity_id = $1 AND drm.member_id = $2
		)
	`

	var exists bool
	if err := db.QueryRowxContext(ctx, query, domainID, memberID).Scan(&exists); err != nil {
		return false, errors.Wrap(fmt.Errorf("failed to check domain role membership"), err)
	}

	return exists, nil
}

func newAuthzedClient(spicedbHost, spicedbPort, spicedbPreSharedKey string) (*authzed.ClientWithExperimental, error) {
	return authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", spicedbHost, spicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(spicedbPreSharedKey),
	)
}

func filterMissingPolicies(ctx context.Context, permClient v1.PermissionsServiceClient, ps []policies.Policy) ([]policies.Policy, []policies.Policy, error) {
	missing := make([]policies.Policy, 0, len(ps))
	existing := make([]policies.Policy, 0)
	for _, p := range ps {
		ok, err := policyExists(ctx, permClient, p)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			existing = append(existing, p)
			continue
		}
		missing = append(missing, p)
	}
	return missing, existing, nil
}

func policyExists(ctx context.Context, permClient v1.PermissionsServiceClient, p policies.Policy) (bool, error) {
	req := &v1.ReadRelationshipsRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{FullyConsistent: true},
		},
		RelationshipFilter: &v1.RelationshipFilter{
			ResourceType:       p.ObjectType,
			OptionalResourceId: p.Object,
			OptionalRelation:   p.Relation,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       p.SubjectType,
				OptionalSubjectId: p.Subject,
			},
		},
		OptionalLimit: 1,
	}

	stream, err := permClient.ReadRelationships(ctx, req)
	if err != nil {
		return false, errors.Wrap(fmt.Errorf("failed to read spicedb relationships"), err)
	}

	for {
		_, err := stream.Recv()
		switch {
		case err == nil:
			return true, nil
		case errors.Contains(err, io.EOF):
			return false, nil
		default:
			return false, errors.Wrap(fmt.Errorf("failed to receive spicedb relationship"), err)
		}
	}
}

func availableActionsAndBuiltInRoles(spicedbSchemaFile string) ([]roles.Action, map[roles.BuiltInRoleName][]roles.Action, error) {
	availableActions, err := spicedbdecoder.GetActionsFromSchema(spicedbSchemaFile, operations.EntityType)
	if err != nil {
		return []roles.Action{}, map[roles.BuiltInRoleName][]roles.Action{}, err
	}

	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		reports.BuiltInRoleAdmin: availableActions,
	}

	return availableActions, builtInRoles, nil
}

func builtInRoleActionStrings(builtInRoles map[roles.BuiltInRoleName][]roles.Action, roleName roles.BuiltInRoleName) ([]string, error) {
	actions, ok := builtInRoles[roleName]
	if !ok {
		return nil, fmt.Errorf("built-in role %q not found", roleName)
	}

	ret := make([]string, 0, len(actions))
	for _, action := range actions {
		ret = append(ret, action.String())
	}

	return ret, nil
}
