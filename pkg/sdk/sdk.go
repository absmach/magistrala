// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	smqerrors "github.com/absmach/magistrala/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"moul.io/http2curl"
)

const (
	// CTJSON represents JSON content type.
	CTJSON ContentType = "application/json"

	// CTJSONSenML represents JSON SenML content type.
	CTJSONSenML ContentType = "application/senml+json"

	// CTBinary represents binary content type.
	CTBinary ContentType = "application/octet-stream"

	// EnabledStatus represents enable status for a client.
	EnabledStatus = "enabled"

	// DisabledStatus represents disabled status for a client.
	DisabledStatus = "disabled"

	BearerPrefix = "Bearer "

	ClientPrefix = "Client "
)

// ContentType represents all possible content types.
type ContentType string

var _ SDK = (*mgSDK)(nil)

var (
	// ErrFailedCreation indicates that entity creation failed.
	ErrFailedCreation = errors.New("failed to create entity in the db")

	// ErrFailedList indicates that entities list failed.
	ErrFailedList = errors.New("failed to list entities")

	// ErrFailedUpdate indicates that entity update failed.
	ErrFailedUpdate = errors.New("failed to update entity")

	// ErrFailedFetch indicates that fetching of entity data failed.
	ErrFailedFetch = errors.New("failed to fetch entity")

	// ErrFailedRemoval indicates that entity removal failed.
	ErrFailedRemoval = errors.New("failed to remove entity")

	// ErrFailedEnable indicates that client enable failed.
	ErrFailedEnable = errors.New("failed to enable client")

	// ErrFailedDisable indicates that client disable failed.
	ErrFailedDisable = errors.New("failed to disable client")

	ErrInvalidJWT = errors.New("invalid JWT")
)

type MessagePageMetadata struct {
	PageMetadata
	Subtopic  string `json:"subtopic,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	// Publishers and DeviceIDs are sent as repeated query parameters, one per
	// value, so entries are transmitted verbatim and never split on a separator.
	Publishers  []string `json:"publishers,omitempty"`
	DeviceIDs   []string `json:"device_ids,omitempty"`
	Comparator  string   `json:"comparator,omitempty"`
	BoolValue   *bool    `json:"vb,omitempty"`
	StringValue string   `json:"vs,omitempty"`
	DataValue   string   `json:"vd,omitempty"`
	From        float64  `json:"from,omitempty"`
	To          float64  `json:"to,omitempty"`
	Aggregation string   `json:"aggregation,omitempty"`
	Interval    string   `json:"interval,omitempty"`
	Value       float64  `json:"value,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
}

// DeviceViewPageMetadata carries the paging and time-range parameters for
// the MG-15 observed-device endpoints, ListGatewayDevices and
// ListDeviceGateways. From and To follow the same convention as
// MessagePageMetadata: left unset (zero), the server applies its own
// default bound rather than running an unbounded aggregation.
//
// From/To are Unix epoch nanoseconds, matching MessagePageMetadata and the
// LastSeen field on the returned stats — the unit the built-in senml/json
// transformers store message time in (see transformers.ToUnixNano). A
// deployment storing a different unit will get an empty page for a defaulted
// window; supply from/to in your own unit explicitly when needed.
type DeviceViewPageMetadata struct {
	Offset uint64  `json:"offset,omitempty"`
	Limit  uint64  `json:"limit,omitempty"`
	From   float64 `json:"from,omitempty"`
	To     float64 `json:"to,omitempty"`
}

type Operator uint8

const (
	OrOp Operator = iota
	AndOp
)

type TagsQuery struct {
	Elements []string
	Operator Operator
}

func ToTagsQuery(s string) TagsQuery {
	switch {
	case strings.Contains(s, "+"):
		elements := strings.Split(s, "+")
		for i := range elements {
			elements[i] = strings.TrimSpace(elements[i])
		}
		return TagsQuery{Elements: elements, Operator: AndOp}
	case strings.Contains(s, ","):
		elements := strings.Split(s, ",")
		for i := range elements {
			elements[i] = strings.TrimSpace(elements[i])
		}
		return TagsQuery{Elements: elements, Operator: OrOp}
	default:
		return TagsQuery{Elements: []string{s}, Operator: OrOp}
	}
}

type PageMetadata struct {
	Total              uint64    `json:"total"`
	Offset             uint64    `json:"offset"`
	Limit              uint64    `json:"limit"`
	Order              string    `json:"order,omitempty"`
	Direction          string    `json:"direction,omitempty"`
	Level              uint64    `json:"level,omitempty"`
	Identity           string    `json:"identity,omitempty"`
	Email              string    `json:"email,omitempty"`
	Username           string    `json:"username,omitempty"`
	LastName           string    `json:"last_name,omitempty"`
	FirstName          string    `json:"first_name,omitempty"`
	Name               string    `json:"name,omitempty"`
	Type               string    `json:"type,omitempty"`
	Metadata           Metadata  `json:"metadata,omitempty"`
	Status             string    `json:"status,omitempty"`
	Action             string    `json:"action,omitempty"`
	Subject            string    `json:"subject,omitempty"`
	Object             string    `json:"object,omitempty"`
	Permission         string    `json:"permission,omitempty"`
	Tags               TagsQuery `json:"tags,omitempty"`
	Owner              string    `json:"owner,omitempty"`
	SharedBy           string    `json:"shared_by,omitempty"`
	Visibility         string    `json:"visibility,omitempty"`
	OwnerID            string    `json:"owner_id,omitempty"`
	Topic              string    `json:"topic,omitempty"`
	Contact            string    `json:"contact,omitempty"`
	State              string    `json:"state,omitempty"`
	ListPermissions    string    `json:"list_perms,omitempty"`
	InvitedBy          string    `json:"invited_by,omitempty"`
	UserID             string    `json:"user_id,omitempty"`
	WorkspaceID        string    `json:"workspace_id,omitempty"`
	Relation           string    `json:"relation,omitempty"`
	Operation          string    `json:"operation,omitempty"`
	From               int64     `json:"from,omitempty"`
	To                 int64     `json:"to,omitempty"`
	WithMetadata       bool      `json:"with_metadata,omitempty"`
	WithAttributes     bool      `json:"with_attributes,omitempty"`
	ID                 string    `json:"id,omitempty"`
	Tree               bool      `json:"tree,omitempty"`
	StartLevel         int64     `json:"start_level,omitempty"`
	EndLevel           int64     `json:"end_level,omitempty"`
	CreatedFrom        time.Time `json:"created_from,omitempty"`
	CreatedTo          time.Time `json:"created_to,omitempty"`
	Dir                string    `json:"dir,omitempty"`
	Tag                string    `json:"tag,omitempty"`
	InputChannel       string    `json:"input_channel,omitempty"`
	RuleID             string    `json:"rule_id,omitempty"`
	ChannelID          string    `json:"channel_id,omitempty"`
	DeviceID           string    `json:"device_id,omitempty"`
	Subtopic           string    `json:"subtopic,omitempty"`
	AssigneeID         string    `json:"assignee_id,omitempty"`
	Severity           uint8     `json:"severity,omitempty"`
	UpdatedBy          string    `json:"updated_by,omitempty"`
	AssignedBy         string    `json:"assigned_by,omitempty"`
	AcknowledgedBy     string    `json:"acknowledged_by,omitempty"`
	ResolvedBy         string    `json:"resolved_by,omitempty"`
	EntityID           string    `json:"entity_id,omitempty"`
	CommonName         string    `json:"common_name,omitempty"`
	Organization       []string  `json:"organization,omitempty"`
	OrganizationalUnit []string  `json:"organizational_unit,omitempty"`
	Country            []string  `json:"country,omitempty"`
	Province           []string  `json:"province,omitempty"`
	Locality           []string  `json:"locality,omitempty"`
	StreetAddress      []string  `json:"street_address,omitempty"`
	PostalCode         []string  `json:"postal_code,omitempty"`
	DNSNames           []string  `json:"dns_names,omitempty"`
	IPAddresses        []string  `json:"ip_addresses,omitempty"`
	EmailAddresses     []string  `json:"email_addresses,omitempty"`
	TTL                string    `json:"ttl,omitempty"`
}

// Credentials represent client credentials: it contains
// "username" which can be a username, generated name;
// and "secret" which can be a password or access token.
type Credentials struct {
	Username string `json:"username,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}

// CertStatus represents the status of a certificate.
type CertStatus int

const (
	CertValid   CertStatus = iota
	CertRevoked CertStatus = iota
	CertUnknown CertStatus = iota
)

const (
	Valid   CertStatus = CertValid
	Revoked CertStatus = CertRevoked
	Unknown CertStatus = CertUnknown
)

// CertType represents CA certificate type.
type CertType int

const (
	RootCA CertType = iota
	IntermediateCA
)

func (c CertType) String() string {
	switch c {
	case RootCA:
		return "root"
	case IntermediateCA:
		return "intermediate"
	default:
		return "unknown"
	}
}

func (c CertStatus) String() string {
	switch c {
	case CertValid:
		return "Valid"
	case CertRevoked:
		return "Revoked"
	default:
		return "Unknown"
	}
}

func (c CertStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// Certificate holds certificate data returned by the certs service SDK.
type Certificate struct {
	SerialNumber string    `json:"serial_number,omitempty"`
	Certificate  string    `json:"certificate,omitempty"`
	Key          string    `json:"key,omitempty"`
	Revoked      bool      `json:"revoked,omitempty"`
	ExpiryTime   time.Time `json:"expiry_time,omitempty"`
	EntityID     string    `json:"entity_id,omitempty"`
	DownloadUrl  string    `json:"-"`
}

// CertificatePage holds a page of certificates.
type CertificatePage struct {
	Total        uint64        `json:"total"`
	Offset       uint64        `json:"offset"`
	Limit        uint64        `json:"limit"`
	Certificates []Certificate `json:"certificates,omitempty"`
}

// CertificateBundle holds CA and certificate data for download.
type CertificateBundle struct {
	CA          []byte `json:"ca"`
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
}

// OCSPResponse holds the OCSP status response for a certificate.
type OCSPResponse struct {
	Status           CertStatus `json:"status"`
	SerialNumber     string     `json:"serial_number"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	ProducedAt       *time.Time `json:"produced_at,omitempty"`
	ThisUpdate       *time.Time `json:"this_update,omitempty"`
	NextUpdate       *time.Time `json:"next_update,omitempty"`
	Certificate      []byte     `json:"certificate,omitempty"`
	IssuerHash       string     `json:"issuer_hash,omitempty"`
	RevocationReason int        `json:"revocation_reason,omitempty"`
}

// Options holds certificate subject options for issuance.
type Options struct {
	CommonName         string   `json:"common_name"`
	Organization       []string `json:"organization"`
	OrganizationalUnit []string `json:"organizational_unit"`
	Country            []string `json:"country"`
	Province           []string `json:"province"`
	Locality           []string `json:"locality"`
	StreetAddress      []string `json:"street_address"`
	PostalCode         []string `json:"postal_code"`
	DnsNames           []string `json:"dns_names"`
}

// CSRMetadata holds metadata for creating a Certificate Signing Request.
type CSRMetadata struct {
	CommonName         string           `json:"common_name"`
	Organization       []string         `json:"organization"`
	OrganizationalUnit []string         `json:"organizational_unit"`
	Country            []string         `json:"country"`
	Province           []string         `json:"province"`
	Locality           []string         `json:"locality"`
	StreetAddress      []string         `json:"street_address"`
	PostalCode         []string         `json:"postal_code"`
	DNSNames           []string         `json:"dns_names"`
	IPAddresses        []string         `json:"ip_addresses"`
	EmailAddresses     []string         `json:"email_addresses"`
	ExtraExtensions    []pkix.Extension `json:"extra_extensions,omitempty"`
}

// CSR holds a Certificate Signing Request in PEM format.
type CSR struct {
	CSR []byte `json:"csr,omitempty"`
}

// SDK contains Magistrala API.
type SDK interface {
	// SendMessage send message to specified channel.
	//
	// example:
	//  ctx := context.Background()
	//  msg := '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
	//  err := sdk.SendMessage(ctx, "workspaceID", "76cc9425-9df0-4b53-99b8-8dabbd3444fc/test", msg, "clientSecret")
	//  fmt.Println(err)
	SendMessage(ctx context.Context, workspaceID, topic, msg, secret string) smqerrors.SDKError

	// SetContentType sets message content type.
	//
	// example:
	//  err := sdk.SetContentType("application/json")
	//  fmt.Println(err)
	SetContentType(ct ContentType) smqerrors.SDKError

	// Health returns service health check.
	//
	// example:
	//  health, _ := sdk.Health("service")
	//  fmt.Println(health)
	Health(service string) (HealthInfo, smqerrors.SDKError)

	// AddBootstrap add bootstrap configuration
	AddBootstrap(ctx context.Context, cfg BootstrapConfig, workspaceID, token string) (string, smqerrors.SDKError)

	// CreateBootstrapProfile creates a bootstrap profile template.
	CreateBootstrapProfile(ctx context.Context, profile BootstrapProfile, workspaceID, token string) (BootstrapProfile, smqerrors.SDKError)

	// ViewBootstrap returns Client Config with given ID belonging to the user identified by the given token.
	ViewBootstrap(ctx context.Context, id, workspaceID, token string) (BootstrapConfig, smqerrors.SDKError)

	// ViewBootstrapProfile returns bootstrap profile with the given ID.
	ViewBootstrapProfile(ctx context.Context, id, workspaceID, token string) (BootstrapProfile, smqerrors.SDKError)

	// UpdateBootstrap updates editable fields of the provided Config.
	UpdateBootstrap(ctx context.Context, cfg BootstrapConfig, workspaceID, token string) smqerrors.SDKError

	// UpdateBootstrapProfile updates editable fields of the provided bootstrap profile and returns the updated profile.
	UpdateBootstrapProfile(ctx context.Context, profile BootstrapProfile, workspaceID, token string) (BootstrapProfile, smqerrors.SDKError)

	// UpdateBootstrapCerts updates bootstrap config certificates.
	UpdateBootstrapCerts(ctx context.Context, id string, clientCert, clientKey, ca string, workspaceID, token string) (BootstrapConfig, smqerrors.SDKError)

	// UpdateBootstrapConnection updates connections performs update of the channel list corresponding Client is connected to.
	UpdateBootstrapConnection(ctx context.Context, id string, channels []string, workspaceID, token string) smqerrors.SDKError

	// RemoveBootstrap removes Config with specified token that belongs to the user identified by the given token.
	RemoveBootstrap(ctx context.Context, id, workspaceID, token string) smqerrors.SDKError

	// RemoveBootstrapProfile removes a bootstrap profile with the given ID.
	RemoveBootstrapProfile(ctx context.Context, id, workspaceID, token string) smqerrors.SDKError

	// Bootstrap returns Config to the Client with provided external ID using external key.
	Bootstrap(ctx context.Context, externalID, externalKey string) (BootstrapConfig, smqerrors.SDKError)

	// BootstrapSecure retrieves a configuration with given external ID and encrypted external key.
	BootstrapSecure(ctx context.Context, externalID, externalKey, cryptoKey string) (BootstrapConfig, smqerrors.SDKError)

	// Bootstraps retrieves a list of managed configs.
	Bootstraps(ctx context.Context, pm PageMetadata, workspaceID, token string) (BootstrapPage, smqerrors.SDKError)

	// BootstrapProfiles retrieves a list of bootstrap profiles.
	BootstrapProfiles(ctx context.Context, pm PageMetadata, workspaceID, token string) (BootstrapProfilesPage, smqerrors.SDKError)

	// Whitelist updates Client bootstrap status with given ID belonging to the user identified by the given token.
	Whitelist(ctx context.Context, clientID string, status BootstrapStatus, workspaceID, token string) smqerrors.SDKError

	// AssignBootstrapProfile assigns a bootstrap profile to the given enrollment.
	AssignBootstrapProfile(ctx context.Context, configID, profileID, workspaceID, token string) smqerrors.SDKError

	// BindBootstrapResources stores resolved binding snapshots for the given enrollment.
	BindBootstrapResources(ctx context.Context, configID string, bindings []BootstrapBindingRequest, workspaceID, token string) smqerrors.SDKError

	// BootstrapBindings lists stored binding snapshots for the given enrollment.
	BootstrapBindings(ctx context.Context, configID, workspaceID, token string) ([]BootstrapBindingSnapshot, smqerrors.SDKError)

	// RefreshBootstrapBindings refreshes stored binding snapshots for the given enrollment.
	RefreshBootstrapBindings(ctx context.Context, configID, workspaceID, token string) smqerrors.SDKError

	// ReadMessages reads messages of specified channel.
	ReadMessages(ctx context.Context, pm MessagePageMetadata, chanID, workspaceID, token string) (MessagesPage, smqerrors.SDKError)

	// ListGatewayDevices lists the devices observed publishing through a
	// gateway on a channel (MG-15): distinct device_id values, each with its
	// last-seen time and message count.
	ListGatewayDevices(ctx context.Context, chanID, publisherID string, pm DeviceViewPageMetadata, workspaceID, token string) (GatewayDevicesPage, smqerrors.SDKError)

	// ListDeviceGateways lists the gateways observed relaying for a device on
	// a channel (MG-15): distinct publisher values, each with its last-seen
	// time and message count.
	ListDeviceGateways(ctx context.Context, chanID, deviceID string, pm DeviceViewPageMetadata, workspaceID, token string) (DeviceGatewaysPage, smqerrors.SDKError)

	// PatchAlarm partially updates an existing alarm.
	PatchAlarm(ctx context.Context, id string, update AlarmUpdate, workspaceID, token string) (Alarm, smqerrors.SDKError)

	// UpdateAlarm updates an existing alarm through the legacy PUT alias.
	// Deprecated: use PatchAlarm.
	UpdateAlarm(ctx context.Context, alarm Alarm, workspaceID, token string) (Alarm, smqerrors.SDKError)

	// ViewAlarm retrieves an alarm by its ID.
	ViewAlarm(ctx context.Context, id, workspaceID, token string) (Alarm, smqerrors.SDKError)

	// ListAlarms retrieves a page of alarms.
	ListAlarms(ctx context.Context, pm PageMetadata, workspaceID, token string) (AlarmsPage, smqerrors.SDKError)

	// DeleteAlarm deletes an alarm.
	DeleteAlarm(ctx context.Context, id, workspaceID, token string) smqerrors.SDKError

	// AddReportConfig creates a new report configuration.
	AddReportConfig(ctx context.Context, cfg ReportConfig, workspaceID, token string) (ReportConfig, smqerrors.SDKError)

	// ViewReportConfig retrieves a report config by its ID.
	ViewReportConfig(ctx context.Context, id, workspaceID, token string) (ReportConfig, smqerrors.SDKError)

	// UpdateReportConfig updates an existing report configuration.
	UpdateReportConfig(ctx context.Context, cfg ReportConfig, workspaceID, token string) (ReportConfig, smqerrors.SDKError)

	// UpdateReportSchedule updates an existing report configuration's schedule.
	UpdateReportSchedule(ctx context.Context, cfg ReportConfig, workspaceID, token string) (ReportConfig, smqerrors.SDKError)

	// RemoveReportConfig deletes a report config.
	RemoveReportConfig(ctx context.Context, id, workspaceID, token string) smqerrors.SDKError

	// ListReportsConfig retrieves a page of report configs.
	ListReportsConfig(ctx context.Context, pm PageMetadata, workspaceID, token string) (ReportConfigPage, smqerrors.SDKError)

	// EnableReportConfig enables a report config.
	EnableReportConfig(ctx context.Context, id, workspaceID, token string) (ReportConfig, smqerrors.SDKError)

	// DisableReportConfig disables a report config.
	DisableReportConfig(ctx context.Context, id, workspaceID, token string) (ReportConfig, smqerrors.SDKError)

	// UpdateReportTemplate updates a report template.
	UpdateReportTemplate(ctx context.Context, cfg ReportConfig, workspaceID, token string) smqerrors.SDKError

	// ViewReportTemplate retrieves a report template.
	ViewReportTemplate(ctx context.Context, id, workspaceID, token string) (ReportTemplate, smqerrors.SDKError)

	// DeleteReportTemplate deletes a report template.
	DeleteReportTemplate(ctx context.Context, id, workspaceID, token string) smqerrors.SDKError

	// GenerateReport generates a report from a configuration.
	GenerateReport(ctx context.Context, config ReportConfig, action ReportAction, workspaceID, token string) (ReportPage, *ReportFile, smqerrors.SDKError)

	// AddRule creates a new rule.
	AddRule(ctx context.Context, r Rule, workspaceID, token string) (Rule, smqerrors.SDKError)

	// ViewRule retrieves a rule by its ID.
	ViewRule(ctx context.Context, id, workspaceID, token string) (Rule, smqerrors.SDKError)

	// UpdateRule updates an existing rule.
	UpdateRule(ctx context.Context, r Rule, workspaceID, token string) (Rule, smqerrors.SDKError)

	// UpdateRuleTags updates an existing rule's tags.
	UpdateRuleTags(ctx context.Context, r Rule, workspaceID, token string) (Rule, smqerrors.SDKError)

	// UpdateRuleSchedule updates an existing rule's schedule.
	UpdateRuleSchedule(ctx context.Context, r Rule, workspaceID, token string) (Rule, smqerrors.SDKError)

	// ListRules retrieves a page of rules.
	ListRules(ctx context.Context, pm PageMetadata, workspaceID, token string) (Page, smqerrors.SDKError)

	// RemoveRule deletes a rule.
	RemoveRule(ctx context.Context, id, workspaceID, token string) smqerrors.SDKError

	// EnableRule enables a rule.
	EnableRule(ctx context.Context, id, workspaceID, token string) (Rule, smqerrors.SDKError)

	// DisableRule disables a rule.
	DisableRule(ctx context.Context, id, workspaceID, token string) (Rule, smqerrors.SDKError)

	// IssueCert issues a certificate for an entity.
	//
	// example:
	//  cert, _ := sdk.IssueCert(context.Background(), "entityID", "8760h", []string{"127.0.0.1"}, sdk.Options{CommonName: "cn"}, "workspaceID", "token")
	IssueCert(ctx context.Context, entityID, ttl string, ipAddrs []string, opts Options, workspaceID, token string) (Certificate, smqerrors.SDKError)

	// RevokeCert revokes a certificate by serial number.
	//
	// example:
	//  err := sdk.RevokeCert(context.Background(), "serialNumber", "workspaceID", "token")
	RevokeCert(ctx context.Context, serialNumber, workspaceID, token string) smqerrors.SDKError

	// RenewCert renews a certificate by serial number.
	//
	// example:
	//  cert, _ := sdk.RenewCert(context.Background(), "serialNumber", "workspaceID", "token")
	RenewCert(ctx context.Context, serialNumber, workspaceID, token string) (Certificate, smqerrors.SDKError)

	// ListCerts lists certificates matching the given metadata filter.
	//
	// example:
	//  page, _ := sdk.ListCerts(context.Background(), sdk.PageMetadata{Limit: 10}, "workspaceID", "token")
	ListCerts(ctx context.Context, pm PageMetadata, workspaceID, token string) (CertificatePage, smqerrors.SDKError)

	// DeleteCert deletes all certificates for the given entity ID.
	//
	// example:
	//  err := sdk.DeleteCert(context.Background(), "entityID", "workspaceID", "token")
	DeleteCert(ctx context.Context, entityID, workspaceID, token string) smqerrors.SDKError

	// ViewCert retrieves a certificate by serial number.
	//
	// example:
	//  cert, _ := sdk.ViewCert(context.Background(), "serialNumber", "workspaceID", "token")
	ViewCert(ctx context.Context, serialNumber, workspaceID, token string) (Certificate, smqerrors.SDKError)

	// OCSP checks the revocation status of a certificate.
	//
	// example:
	//  resp, _ := sdk.OCSP(context.Background(), "serialNumber", "")
	OCSP(ctx context.Context, serialNumber, cert string) (OCSPResponse, smqerrors.SDKError)

	// ViewCA views the signing CA certificate.
	//
	// example:
	//  cert, _ := sdk.ViewCA(context.Background())
	ViewCA(ctx context.Context) (Certificate, smqerrors.SDKError)

	// DownloadCA downloads the signing CA certificate bundle.
	//
	// example:
	//  bundle, _ := sdk.DownloadCA(context.Background())
	DownloadCA(ctx context.Context) (CertificateBundle, smqerrors.SDKError)

	// IssueFromCSR issues a certificate from a provided CSR.
	//
	// example:
	//  cert, _ := sdk.IssueFromCSR(context.Background(), "entityID", "8760h", csrPEM, "workspaceID", "token")
	IssueFromCSR(ctx context.Context, entityID, ttl, csr, workspaceID, token string) (Certificate, smqerrors.SDKError)

	// IssueFromCSRInternal issues a certificate from a CSR using agent authentication.
	//
	// example:
	//  cert, _ := sdk.IssueFromCSRInternal(context.Background(), "entityID", "8760h", csrPEM, "agentToken")
	IssueFromCSRInternal(ctx context.Context, entityID, ttl, csr, token string) (Certificate, smqerrors.SDKError)

	// GenerateCRL generates a Certificate Revocation List.
	//
	// example:
	//  crl, _ := sdk.GenerateCRL(context.Background())
	GenerateCRL(ctx context.Context) ([]byte, smqerrors.SDKError)

	// RevokeAll revokes all certificates for an entity ID.
	//
	// example:
	//  err := sdk.RevokeAll(context.Background(), "entityID", "workspaceID", "token")
	RevokeAll(ctx context.Context, entityID, workspaceID, token string) smqerrors.SDKError

	// EntityID gets the entity ID for a certificate by serial number.
	//
	// example:
	//  id, _ := sdk.EntityID(context.Background(), "serialNumber", "workspaceID", "token")
	EntityID(ctx context.Context, serialNumber, workspaceID, token string) (string, smqerrors.SDKError)

	// CreateCSR creates a Certificate Signing Request from metadata and a private key.
	//
	// example:
	//  csr, _ := sdk.CreateCSR(context.Background(), metadata, privateKeyBytes)
	CreateCSR(ctx context.Context, metadata CSRMetadata, privKey any) (CSR, smqerrors.SDKError)
}

type mgSDK struct {
	certsURL       string
	httpAdapterURL string
	HostURL        string
	bootstrapURL   string
	readersURL     string
	alarmsURL      string
	reportsURL     string
	rulesEngineURL string

	msgContentType ContentType
	client         *http.Client
	curlFlag       bool
	roles          bool
}

// Config contains sdk configuration parameters.
type Config struct {
	CertsURL       string
	HTTPAdapterURL string
	HostURL        string
	BootstrapURL   string
	ReaderURL      string
	AlarmsURL      string
	ReportsURL     string
	RulesEngineURL string

	MsgContentType  ContentType
	TLSVerification bool
	CurlFlag        bool
	Roles           bool
}

// NewSDK returns new magistrala SDK instance.
func NewSDK(conf Config) SDK {
	return &mgSDK{
		certsURL:       conf.CertsURL,
		httpAdapterURL: conf.HTTPAdapterURL,
		HostURL:        conf.HostURL,
		bootstrapURL:   conf.BootstrapURL,
		readersURL:     conf.ReaderURL,
		alarmsURL:      conf.AlarmsURL,
		reportsURL:     conf.ReportsURL,
		rulesEngineURL: conf.RulesEngineURL,

		msgContentType: conf.MsgContentType,
		client: &http.Client{Transport: otelhttp.NewTransport(&http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !conf.TLSVerification,
			},
			IdleConnTimeout: 90 * time.Second,
		})},
		curlFlag: conf.CurlFlag,
		roles:    conf.Roles,
	}
}

// processRequest creates and send a new HTTP request, and checks for errors in the HTTP response.
// It then returns the response headers, the response body, and the associated error(s) (if any).
func (sdk mgSDK) processRequest(ctx context.Context, method, reqUrl, token string, data []byte, headers map[string]string, expectedRespCodes ...int) (http.Header, []byte, smqerrors.SDKError) {
	if sdk.roles {
		reqUrl = fmt.Sprintf("%s?roles=%v", reqUrl, true)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqUrl, bytes.NewReader(data))
	if err != nil {
		return make(http.Header), []byte{}, smqerrors.NewSDKError(err)
	}

	// Sets a default value for the Content-Type.
	// Overridden if Content-Type is passed in the headers arguments.
	req.Header.Add("Content-Type", string(CTJSON))

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	if token != "" {
		if !strings.Contains(token, ClientPrefix) {
			token = BearerPrefix + token
		}
		req.Header.Set("Authorization", token)
	}

	if sdk.curlFlag {
		curlCommand, err := http2curl.GetCurlCommand(req)
		if err != nil {
			return nil, nil, smqerrors.NewSDKError(err)
		}
		log.Println(curlCommand.String())
	}

	resp, err := sdk.client.Do(req)
	if err != nil {
		var opErr *net.OpError
		switch {
		case errors.Is(err, syscall.ECONNRESET):
			return make(http.Header), []byte{}, smqerrors.NewSDKError(fmt.Errorf("request failed: connection reset by peer: %w", err))
		case errors.As(err, &opErr):
			return make(http.Header), []byte{}, smqerrors.NewSDKError(fmt.Errorf("request failed: network error (%s): %w", opErr.Op, err))
		case errors.Is(err, io.EOF):
			return make(http.Header), []byte{}, smqerrors.NewSDKError(fmt.Errorf("request failed: connection closed unexpectedly: %w", err))
		default:
			return make(http.Header), []byte{}, smqerrors.NewSDKError(fmt.Errorf("request failed: %w", err))
		}
	}
	defer resp.Body.Close()

	sdkErr := smqerrors.CheckError(resp, expectedRespCodes...)
	if sdkErr != nil {
		return make(http.Header), []byte{}, sdkErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return make(http.Header), []byte{}, smqerrors.NewSDKError(err)
	}

	return resp.Header, body, nil
}

func (sdk mgSDK) withQueryParams(baseURL, endpoint string, pm PageMetadata) (string, error) {
	q, err := pm.query()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s?%s", baseURL, endpoint, q), nil
}

func (pm PageMetadata) query() (string, error) {
	q := url.Values{}
	addStringSlice := func(key string, values []string) {
		for _, value := range values {
			if value == "" {
				continue
			}
			q.Add(key, value)
		}
	}

	if pm.Offset != 0 {
		q.Add("offset", strconv.FormatUint(pm.Offset, 10))
	}
	if pm.Limit != 0 {
		q.Add("limit", strconv.FormatUint(pm.Limit, 10))
	}
	if pm.Total != 0 {
		q.Add("total", strconv.FormatUint(pm.Total, 10))
	}
	if pm.Order != "" {
		q.Add("order", pm.Order)
	}
	if pm.Direction != "" {
		q.Add("dir", pm.Direction)
	}
	if pm.Level != 0 {
		q.Add("level", strconv.FormatUint(pm.Level, 10))
	}
	if pm.Email != "" {
		q.Add("email", pm.Email)
	}
	if pm.Identity != "" {
		q.Add("identity", pm.Identity)
	}
	if pm.Username != "" {
		q.Add("username", pm.Username)
	}
	if pm.FirstName != "" {
		q.Add("first_name", pm.FirstName)
	}
	if pm.LastName != "" {
		q.Add("last_name", pm.LastName)
	}
	if pm.Name != "" {
		q.Add("name", pm.Name)
	}
	if pm.ID != "" {
		q.Add("id", pm.ID)
	}
	if pm.Type != "" {
		q.Add("type", pm.Type)
	}
	if pm.Visibility != "" {
		q.Add("visibility", pm.Visibility)
	}
	if pm.Status != "" {
		q.Add("status", pm.Status)
	}
	if pm.Metadata != nil {
		md, err := json.Marshal(pm.Metadata)
		if err != nil {
			return "", smqerrors.NewSDKError(err)
		}
		q.Add("metadata", string(md))
	}
	if pm.Action != "" {
		q.Add("action", pm.Action)
	}
	if pm.Subject != "" {
		q.Add("subject", pm.Subject)
	}
	if pm.Object != "" {
		q.Add("object", pm.Object)
	}
	if len(pm.Tags.Elements) > 0 {
		switch pm.Tags.Operator {
		case AndOp:
			str := strings.Join(pm.Tags.Elements, "-")
			q.Add("tags", str)
		default:
			str := strings.Join(pm.Tags.Elements, ",")
			q.Add("tags", str)
		}
	}
	if pm.Owner != "" {
		q.Add("owner", pm.Owner)
	}
	if pm.SharedBy != "" {
		q.Add("shared_by", pm.SharedBy)
	}
	if pm.Topic != "" {
		q.Add("topic", pm.Topic)
	}
	if pm.Contact != "" {
		q.Add("contact", pm.Contact)
	}
	if pm.State != "" {
		q.Add("state", pm.State)
	}
	if pm.Permission != "" {
		q.Add("permission", pm.Permission)
	}
	if pm.ListPermissions != "" {
		q.Add("list_perms", pm.ListPermissions)
	}
	if pm.InvitedBy != "" {
		q.Add("invited_by", pm.InvitedBy)
	}
	if pm.UserID != "" {
		q.Add("user_id", pm.UserID)
	}
	if pm.WorkspaceID != "" {
		q.Add("workspace_id", pm.WorkspaceID)
	}
	if pm.Relation != "" {
		q.Add("relation", pm.Relation)
	}
	if pm.Operation != "" {
		q.Add("operation", pm.Operation)
	}
	if pm.From != 0 {
		q.Add("from", strconv.FormatInt(pm.From, 10))
	}
	if pm.To != 0 {
		q.Add("to", strconv.FormatInt(pm.To, 10))
	}
	if !pm.CreatedFrom.IsZero() {
		q.Add("created_from", pm.CreatedFrom.Format(time.RFC3339))
	}
	if !pm.CreatedTo.IsZero() {
		q.Add("created_to", pm.CreatedTo.Format(time.RFC3339))
	}
	q.Add("with_attributes", strconv.FormatBool(pm.WithAttributes))
	q.Add("with_metadata", strconv.FormatBool(pm.WithMetadata))
	if pm.EntityID != "" {
		q.Add("entity_id", pm.EntityID)
	}
	if pm.CommonName != "" {
		q.Add("common_name", pm.CommonName)
	}
	addStringSlice("organization", pm.Organization)
	addStringSlice("organizational_unit", pm.OrganizationalUnit)
	addStringSlice("country", pm.Country)
	addStringSlice("province", pm.Province)
	addStringSlice("locality", pm.Locality)
	addStringSlice("street_address", pm.StreetAddress)
	addStringSlice("postal_code", pm.PostalCode)
	addStringSlice("dns_names", pm.DNSNames)
	addStringSlice("ip_addresses", pm.IPAddresses)
	addStringSlice("email_addresses", pm.EmailAddresses)
	if pm.TTL != "" {
		q.Add("ttl", pm.TTL)
	}

	return q.Encode(), nil
}
