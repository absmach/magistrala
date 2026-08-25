# Magistrala Go SDK

Go SDK, a Go driver for Magistrala HTTP API.

Identity and authorization (workspaces, groups, users, clients, channels) are now served by [Atom](https://github.com/absmach/magistrala/tree/main/pkg/atom) over GraphQL and are out of scope for this SDK. This SDK covers the services that still expose a Magistrala HTTP API: messaging, bootstrap provisioning, certificates, alarms, reports, and the rules engine.

## Installation

Import `"github.com/absmach/magistrala/pkg/sdk"` in your Go package.

```go
import "github.com/absmach/magistrala/pkg/sdk"
```

You can check [Magistrala CLI](https://github.com/absmach/magistrala/tree/main/cli) as an example of SDK usage.

## Quick Start

```go
import (
    "context"
    "fmt"
    "github.com/absmach/magistrala/pkg/sdk"
)

func main() {
    conf := sdk.Config{
        CertsURL:       "http://localhost:9019",
        HTTPAdapterURL: "http://localhost:8008",
        BootstrapURL:   "http://localhost:9013",
        ReaderURL:      "http://localhost:9011",
        AlarmsURL:      "http://localhost:9021",
        ReportsURL:     "http://localhost:9022",
        RulesEngineURL: "http://localhost:9023",
        HostURL:        "http://localhost",
    }

    // Create SDK instance
    smqsdk := sdk.NewSDK(conf)

    ctx := context.Background()

    // Send a message to a channel (token/secret obtained from Atom)
    if err := smqsdk.SendMessage(ctx, workspaceID, "channelID", `{"value":42}`, secret); err != nil {
        fmt.Printf("Error sending message: %v\n", err)
        return
    }
}
```

## API Reference

### Configuration

```go
type Config struct {
    CertsURL        string
    HTTPAdapterURL  string
    HostURL         string
    BootstrapURL    string
    ReaderURL       string
    AlarmsURL       string
    ReportsURL      string
    RulesEngineURL  string
    MsgContentType  ContentType
    TLSVerification bool
    CurlFlag        bool
    Roles           bool
}

func NewSDK(conf Config) SDK
```

### Messaging

```go
// Send message to channel
SendMessage(ctx context.Context, workspaceID, topic, msg, secret string) errors.SDKError

// Read messages from a channel
ReadMessages(ctx context.Context, pm MessagePageMetadata, chanID, workspaceID, token string) (MessagesPage, errors.SDKError)

// Set message content type
SetContentType(ct ContentType) errors.SDKError

// List devices observed publishing through a gateway, and vice versa
ListGatewayDevices(ctx context.Context, chanID, publisherID string, pm DeviceViewPageMetadata, workspaceID, token string) (GatewayDevicesPage, errors.SDKError)
ListDeviceGateways(ctx context.Context, chanID, deviceID string, pm DeviceViewPageMetadata, workspaceID, token string) (DeviceGatewaysPage, errors.SDKError)
```

### Bootstrap Management

```go
// Manage bootstrap configs
AddBootstrap(ctx context.Context, cfg BootstrapConfig, workspaceID, token string) (string, errors.SDKError)
ViewBootstrap(ctx context.Context, id, workspaceID, token string) (BootstrapConfig, errors.SDKError)
UpdateBootstrap(ctx context.Context, cfg BootstrapConfig, workspaceID, token string) errors.SDKError
UpdateBootstrapCerts(ctx context.Context, id string, clientCert, clientKey, ca string, workspaceID, token string) (BootstrapConfig, errors.SDKError)
RemoveBootstrap(ctx context.Context, id, workspaceID, token string) errors.SDKError
Bootstraps(ctx context.Context, pm PageMetadata, workspaceID, token string) (BootstrapPage, errors.SDKError)

// Device-facing bootstrap retrieval
Bootstrap(ctx context.Context, externalID, externalKey string) (BootstrapConfig, errors.SDKError)
BootstrapSecure(ctx context.Context, externalID, externalKey, cryptoKey string) (BootstrapConfig, errors.SDKError)
Whitelist(ctx context.Context, clientID string, status BootstrapStatus, workspaceID, token string) errors.SDKError

// Bootstrap profiles
CreateBootstrapProfile(ctx context.Context, profile BootstrapProfile, workspaceID, token string) (BootstrapProfile, errors.SDKError)
ViewBootstrapProfile(ctx context.Context, id, workspaceID, token string) (BootstrapProfile, errors.SDKError)
UpdateBootstrapProfile(ctx context.Context, profile BootstrapProfile, workspaceID, token string) (BootstrapProfile, errors.SDKError)
RemoveBootstrapProfile(ctx context.Context, id, workspaceID, token string) errors.SDKError
BootstrapProfiles(ctx context.Context, pm PageMetadata, workspaceID, token string) (BootstrapProfilesPage, errors.SDKError)
AssignBootstrapProfile(ctx context.Context, configID, profileID, workspaceID, token string) errors.SDKError

// Bootstrap enrollments
BindBootstrapResources(ctx context.Context, configID string, bindings []BootstrapBindingRequest, workspaceID, token string) errors.SDKError
BootstrapBindings(ctx context.Context, configID, workspaceID, token string) ([]BootstrapBindingSnapshot, errors.SDKError)
RefreshBootstrapBindings(ctx context.Context, configID, workspaceID, token string) errors.SDKError
```

### Alarms

```go
UpdateAlarm(ctx context.Context, alarm Alarm, workspaceID, token string) (Alarm, errors.SDKError)
ViewAlarm(ctx context.Context, id, workspaceID, token string) (Alarm, errors.SDKError)
ListAlarms(ctx context.Context, pm PageMetadata, workspaceID, token string) (AlarmsPage, errors.SDKError)
DeleteAlarm(ctx context.Context, id, workspaceID, token string) errors.SDKError
```

### Reports

```go
AddReportConfig(ctx context.Context, cfg ReportConfig, workspaceID, token string) (ReportConfig, errors.SDKError)
ViewReportConfig(ctx context.Context, id, workspaceID, token string) (ReportConfig, errors.SDKError)
UpdateReportConfig(ctx context.Context, cfg ReportConfig, workspaceID, token string) (ReportConfig, errors.SDKError)
UpdateReportSchedule(ctx context.Context, cfg ReportConfig, workspaceID, token string) (ReportConfig, errors.SDKError)
RemoveReportConfig(ctx context.Context, id, workspaceID, token string) errors.SDKError
ListReportsConfig(ctx context.Context, pm PageMetadata, workspaceID, token string) (ReportConfigPage, errors.SDKError)
EnableReportConfig(ctx context.Context, id, workspaceID, token string) (ReportConfig, errors.SDKError)
DisableReportConfig(ctx context.Context, id, workspaceID, token string) (ReportConfig, errors.SDKError)
GenerateReport(ctx context.Context, config ReportConfig, action ReportAction, workspaceID, token string) (ReportPage, *ReportFile, errors.SDKError)

// Report templates
UpdateReportTemplate(ctx context.Context, cfg ReportConfig, workspaceID, token string) errors.SDKError
ViewReportTemplate(ctx context.Context, id, workspaceID, token string) (ReportTemplate, errors.SDKError)
DeleteReportTemplate(ctx context.Context, id, workspaceID, token string) errors.SDKError
```

### Rules Engine

```go
AddRule(ctx context.Context, r Rule, workspaceID, token string) (Rule, errors.SDKError)
ViewRule(ctx context.Context, id, workspaceID, token string) (Rule, errors.SDKError)
UpdateRule(ctx context.Context, r Rule, workspaceID, token string) (Rule, errors.SDKError)
UpdateRuleTags(ctx context.Context, r Rule, workspaceID, token string) (Rule, errors.SDKError)
UpdateRuleSchedule(ctx context.Context, r Rule, workspaceID, token string) (Rule, errors.SDKError)
ListRules(ctx context.Context, pm PageMetadata, workspaceID, token string) (Page, errors.SDKError)
RemoveRule(ctx context.Context, id, workspaceID, token string) errors.SDKError
EnableRule(ctx context.Context, id, workspaceID, token string) (Rule, errors.SDKError)
DisableRule(ctx context.Context, id, workspaceID, token string) (Rule, errors.SDKError)
```

### Certificate Management

```go
// Issue certificate for mTLS
IssueCert(ctx context.Context, entityID, ttl string, ipAddrs []string, opts Options, workspaceID, token string) (Certificate, errors.SDKError)

// View certificates
ViewCert(ctx context.Context, serialNumber, workspaceID, token string) (Certificate, errors.SDKError)
ListCerts(ctx context.Context, pm PageMetadata, workspaceID, token string) (CertificatePage, errors.SDKError)

// Revoke and renew certificates
RevokeCert(ctx context.Context, serialNumber, workspaceID, token string) errors.SDKError
RevokeAll(ctx context.Context, entityID, workspaceID, token string) errors.SDKError
RenewCert(ctx context.Context, serialNumber, workspaceID, token string) (Certificate, errors.SDKError)
DeleteCert(ctx context.Context, entityID, workspaceID, token string) errors.SDKError

// CA and CSR operations
ViewCA(ctx context.Context) (Certificate, errors.SDKError)
DownloadCA(ctx context.Context) (CertificateBundle, errors.SDKError)
IssueFromCSR(ctx context.Context, entityID, ttl, csr, workspaceID, token string) (Certificate, errors.SDKError)
IssueFromCSRInternal(ctx context.Context, entityID, ttl, csr, token string) (Certificate, errors.SDKError)
CreateCSR(ctx context.Context, metadata CSRMetadata, privKey any) (CSR, errors.SDKError)
GenerateCRL(ctx context.Context) ([]byte, errors.SDKError)
OCSP(ctx context.Context, serialNumber, cert string) (OCSPResponse, errors.SDKError)
EntityID(ctx context.Context, serialNumber, workspaceID, token string) (string, errors.SDKError)
```

### Health Check

```go
// Service health check (supports "certs" and "fluxmq")
Health(service string) (HealthInfo, errors.SDKError)
```

## Examples

### Bootstrap a device

```go
ctx := context.Background()

cfg := sdk.BootstrapConfig{
    ExternalID:  "external-id",
    ExternalKey: "external-key",
    Channels:    []string{channelID},
    Name:        "My Device",
}
id, err := smqsdk.AddBootstrap(ctx, cfg, workspaceID, token)
```

### Issue a certificate

```go
opts := sdk.Options{CommonName: "my-device"}
cert, err := smqsdk.IssueCert(ctx, entityID, "8760h", nil, opts, workspaceID, token)
```

### Send and read messages

```go
err := smqsdk.SendMessage(ctx, workspaceID, "channelID", `{"value":42}`, secret)

pm := sdk.MessagePageMetadata{PageMetadata: sdk.PageMetadata{Offset: 0, Limit: 10}}
messages, err := smqsdk.ReadMessages(ctx, pm, "channelID", workspaceID, token)
```
