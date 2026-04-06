# Magistrala Go SDK

Go SDK, a Go driver for Magistrala HTTP API.

Provides comprehensive functionality for system administration (provisioning), messaging, user management, domain management, groups, channels, clients, certificates, invitations, and journal operations.

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
        UsersURL:       "http://localhost:9002",
        ClientsURL:     "http://localhost:9000",
        ChannelsURL:    "http://localhost:9001",
        DomainsURL:     "http://localhost:8189",
        HTTPAdapterURL: "http://localhost:8008",
        CertsURL:       "http://localhost:9019",
        JournalURL:     "http://localhost:9021",
        HostURL:        "http://localhost",
    }

    // Create SDK instance
    smqsdk := sdk.NewSDK(conf)

    ctx := context.Background()

    // Create user
    user := sdk.User{
        Name: "John Doe",
        Email: "john.doe@example.com",
        Credentials: sdk.Credentials{
            Username: "john.doe",
            Secret:   "12345678",
        },
    }
    user, err := smqsdk.CreateUser(ctx, user, "")
    if err != nil {
        fmt.Printf("Error creating user: %v\n", err)
        return
    }

    // Create token
    login := sdk.Login{
        Identity: "john.doe",
        Secret:   "12345678",
    }
    token, err := smqsdk.CreateToken(ctx, login)
    if err != nil {
        fmt.Printf("Error creating token: %v\n", err)
        return
    }

    fmt.Printf("User created: %+v\n", user)
    fmt.Printf("Token: %s\n", token.AccessToken)
}
```

## API Reference

### Configuration

```go
type Config struct {
    CertsURL        string
    HTTPAdapterURL  string
    ClientsURL      string
    UsersURL        string
    GroupsURL       string
    ChannelsURL     string
    DomainsURL      string
    JournalURL      string
    HostURL         string
    MsgContentType  ContentType
    TLSVerification bool
    CurlFlag        bool
    Roles           bool
}

func NewSDK(conf Config) SDK
```

### User Management

```go
// Create a new user
CreateUser(ctx context.Context, user User, token string) (User, errors.SDKError)

// Get user by ID
User(ctx context.Context, id, token string) (User, errors.SDKError)

// Get current user profile
UserProfile(ctx context.Context, token string) (User, errors.SDKError)

// List users with pagination
Users(ctx context.Context, pm PageMetadata, token string) (UsersPage, errors.SDKError)

// Search users
SearchUsers(ctx context.Context, pm PageMetadata, token string) (UsersPage, errors.SDKError)

// Update user information
UpdateUser(ctx context.Context, user User, token string) (User, errors.SDKError)
UpdateUserEmail(ctx context.Context, user User, token string) (User, errors.SDKError)
UpdateUserTags(ctx context.Context, user User, token string) (User, errors.SDKError)
UpdateUsername(ctx context.Context, user User, token string) (User, errors.SDKError)
UpdateProfilePicture(ctx context.Context, user User, token string) (User, errors.SDKError)
UpdateUserRole(ctx context.Context, user User, token string) (User, errors.SDKError)

// Password management
UpdatePassword(ctx context.Context, oldPass, newPass, token string) (User, errors.SDKError)
ResetPasswordRequest(ctx context.Context, email string) errors.SDKError
ResetPassword(ctx context.Context, password, confPass, token string) errors.SDKError

// User status management
EnableUser(ctx context.Context, id, token string) (User, errors.SDKError)
DisableUser(ctx context.Context, id, token string) (User, errors.SDKError)
DeleteUser(ctx context.Context, id, token string) errors.SDKError
```

### Authentication

```go
// Create authentication token
CreateToken(ctx context.Context, lt Login) (Token, errors.SDKError)

// Refresh authentication token
RefreshToken(ctx context.Context, token string) (Token, errors.SDKError)
```

### Domain Management

```go
// Create domain
CreateDomain(ctx context.Context, d Domain, token string) (Domain, errors.SDKError)

// Get domain information
Domain(ctx context.Context, domainID, token string) (Domain, errors.SDKError)

// List domains
Domains(ctx context.Context, pm PageMetadata, token string) (DomainsPage, errors.SDKError)

// Update domain
UpdateDomain(ctx context.Context, d Domain, token string) (Domain, errors.SDKError)

// Domain status management
EnableDomain(ctx context.Context, domainID, token string) errors.SDKError
DisableDomain(ctx context.Context, domainID, token string) errors.SDKError
FreezeDomain(ctx context.Context, domainID, token string) errors.SDKError

// Domain roles management
CreateDomainRole(ctx context.Context, id string, rq RoleReq, token string) (Role, errors.SDKError)
DomainRoles(ctx context.Context, id string, pm PageMetadata, token string) (RolesPage, errors.SDKError)
DomainRole(ctx context.Context, id, roleID, token string) (Role, errors.SDKError)
UpdateDomainRole(ctx context.Context, id, roleID, newName string, token string) (Role, errors.SDKError)
DeleteDomainRole(ctx context.Context, id, roleID, token string) errors.SDKError

// Domain role actions management
AddDomainRoleActions(ctx context.Context, id, roleID string, actions []string, token string) ([]string, errors.SDKError)
DomainRoleActions(ctx context.Context, id, roleID string, token string) ([]string, errors.SDKError)
RemoveDomainRoleActions(ctx context.Context, id, roleID string, actions []string, token string) errors.SDKError
RemoveAllDomainRoleActions(ctx context.Context, id, roleID, token string) errors.SDKError
AvailableDomainRoleActions(ctx context.Context, token string) ([]string, errors.SDKError)

// Domain role members management
AddDomainRoleMembers(ctx context.Context, id, roleID string, members []string, token string) ([]string, errors.SDKError)
DomainRoleMembers(ctx context.Context, id, roleID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)
RemoveDomainRoleMembers(ctx context.Context, id, roleID string, members []string, token string) errors.SDKError
RemoveAllDomainRoleMembers(ctx context.Context, id, roleID, token string) errors.SDKError
ListDomainMembers(ctx context.Context, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Client Management

```go
// Create clients
CreateClient(ctx context.Context, client Client, domainID, token string) (Client, errors.SDKError)
CreateClients(ctx context.Context, client []Client, domainID, token string) ([]Client, errors.SDKError)

// Get client information
Client(ctx context.Context, id, domainID, token string) (Client, errors.SDKError)
Clients(ctx context.Context, pm PageMetadata, domainID, token string) (ClientsPage, errors.SDKError)

// Update clients
UpdateClient(ctx context.Context, client Client, domainID, token string) (Client, errors.SDKError)
UpdateClientTags(ctx context.Context, client Client, domainID, token string) (Client, errors.SDKError)
UpdateClientSecret(ctx context.Context, id, secret, domainID, token string) (Client, errors.SDKError)

// Client status management
EnableClient(ctx context.Context, id, domainID, token string) (Client, errors.SDKError)
DisableClient(ctx context.Context, id, domainID, token string) (Client, errors.SDKError)
DeleteClient(ctx context.Context, id, domainID, token string) errors.SDKError

// Client hierarchy management
SetClientParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError
RemoveClientParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

// Client roles management
CreateClientRole(ctx context.Context, id, domainID string, rq RoleReq, token string) (Role, errors.SDKError)
ClientRoles(ctx context.Context, id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)
ClientRole(ctx context.Context, id, roleID, domainID, token string) (Role, errors.SDKError)
UpdateClientRole(ctx context.Context, id, roleID, newName, domainID string, token string) (Role, errors.SDKError)
DeleteClientRole(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

// Client role actions management
AddClientRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError)
ClientRoleActions(ctx context.Context, id, roleID, domainID string, token string) ([]string, errors.SDKError)
RemoveClientRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) errors.SDKError
RemoveAllClientRoleActions(ctx context.Context, id, roleID, domainID, token string) errors.SDKError
AvailableClientRoleActions(ctx context.Context, domainID, token string) ([]string, errors.SDKError)

// Client role members management
AddClientRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError)
ClientRoleMembers(ctx context.Context, id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)
RemoveClientRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) errors.SDKError
RemoveAllClientRoleMembers(ctx context.Context, id, roleID, domainID, token string) errors.SDKError
ListClientMembers(ctx context.Context, clientID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Channel Management

```go
// Create channels
CreateChannel(ctx context.Context, channel Channel, domainID, token string) (Channel, errors.SDKError)
CreateChannels(ctx context.Context, channels []Channel, domainID, token string) ([]Channel, errors.SDKError)

// Get channel information
Channel(ctx context.Context, id, domainID, token string) (Channel, errors.SDKError)
Channels(ctx context.Context, pm PageMetadata, domainID, token string) (ChannelsPage, errors.SDKError)

// Update channels
UpdateChannel(ctx context.Context, channel Channel, domainID, token string) (Channel, errors.SDKError)
UpdateChannelTags(ctx context.Context, c Channel, domainID, token string) (Channel, errors.SDKError)

// Channel status management
EnableChannel(ctx context.Context, id, domainID, token string) (Channel, errors.SDKError)
DisableChannel(ctx context.Context, id, domainID, token string) (Channel, errors.SDKError)
DeleteChannel(ctx context.Context, id, domainID, token string) errors.SDKError

// Channel hierarchy management
SetChannelParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError
RemoveChannelParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

// Channel connections
Connect(ctx context.Context, conn Connection, domainID, token string) errors.SDKError
Disconnect(ctx context.Context, conn Connection, domainID, token string) errors.SDKError
ConnectClients(ctx context.Context, channelID string, clientIDs, connTypes []string, domainID, token string) errors.SDKError
DisconnectClients(ctx context.Context, channelID string, clientIDs, connTypes []string, domainID, token string) errors.SDKError

// List channel members
ListChannelMembers(ctx context.Context, channelID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Group Management

```go
// Create group
CreateGroup(ctx context.Context, group Group, domainID, token string) (Group, errors.SDKError)

// Get group information
Group(ctx context.Context, id, domainID, token string) (Group, errors.SDKError)
Groups(ctx context.Context, pm PageMetadata, domainID, token string) (GroupsPage, errors.SDKError)

// Update groups
UpdateGroup(ctx context.Context, group Group, domainID, token string) (Group, errors.SDKError)
UpdateGroupTags(ctx context.Context, group Group, domainID, token string) (Group, errors.SDKError)

// Group status management
EnableGroup(ctx context.Context, id, domainID, token string) (Group, errors.SDKError)
DisableGroup(ctx context.Context, id, domainID, token string) (Group, errors.SDKError)
DeleteGroup(ctx context.Context, id, domainID, token string) errors.SDKError

// Group hierarchy management
SetGroupParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError
RemoveGroupParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError
AddChildren(ctx context.Context, id, domainID string, groupIDs []string, token string) errors.SDKError
RemoveChildren(ctx context.Context, id, domainID string, groupIDs []string, token string) errors.SDKError
RemoveAllChildren(ctx context.Context, id, domainID, token string) errors.SDKError
Children(ctx context.Context, id, domainID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError)
Hierarchy(ctx context.Context, id, domainID string, pm PageMetadata, token string) (GroupsHierarchyPage, errors.SDKError)

// Group roles management
CreateGroupRole(ctx context.Context, id, domainID string, rq RoleReq, token string) (Role, errors.SDKError)
GroupRoles(ctx context.Context, id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)
GroupRole(ctx context.Context, id, roleID, domainID, token string) (Role, errors.SDKError)
UpdateGroupRole(ctx context.Context, id, roleID, newName, domainID string, token string) (Role, errors.SDKError)
DeleteGroupRole(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

// Group role actions management
AddGroupRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError)
GroupRoleActions(ctx context.Context, id, roleID, domainID string, token string) ([]string, errors.SDKError)
RemoveGroupRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) errors.SDKError
RemoveAllGroupRoleActions(ctx context.Context, id, roleID, domainID, token string) errors.SDKError
AvailableGroupRoleActions(ctx context.Context, id, token string) ([]string, errors.SDKError)

// Group role members management
AddGroupRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError)
GroupRoleMembers(ctx context.Context, id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)
RemoveGroupRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) errors.SDKError
RemoveAllGroupRoleMembers(ctx context.Context, id, roleID, domainID, token string) errors.SDKError
ListGroupMembers(ctx context.Context, groupID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Certificate Management

```go
// Issue certificate for mTLS
IssueCert(ctx context.Context, clientID, validity, domainID, token string) (Cert, errors.SDKError)

// View certificate
ViewCert(ctx context.Context, certID, domainID, token string) (Cert, errors.SDKError)

// View certificates by client
ViewCertByClient(ctx context.Context, clientID, domainID, token string) (CertSerials, errors.SDKError)

// Revoke certificates
RevokeCert(ctx context.Context, certID, domainID, token string) (time.Time, errors.SDKError)
RevokeAllCerts(ctx context.Context, clientID, domainID, token string) (time.Time, errors.SDKError)
```

### Invitation Management

```go
// Send invitation
SendInvitation(ctx context.Context, invitation Invitation, token string) error

// List invitations
Invitations(ctx context.Context, pm PageMetadata, token string) (InvitationPage, error)
DomainInvitations(ctx context.Context, pm PageMetadata, token, domainID string) (InvitationPage, error)

// Manage invitations
AcceptInvitation(ctx context.Context, domainID, token string) error
RejectInvitation(ctx context.Context, domainID, token string) error
DeleteInvitation(ctx context.Context, userID, domainID, token string) error
```

### Journal Management

```go
// Get journal logs
Journal(ctx context.Context, entityType, entityID, domainID string, pm PageMetadata, token string) (JournalsPage, error)
```

### Messaging

```go
// Send message to channel
SendMessage(ctx context.Context, domainID, topic, msg, secret string) errors.SDKError

// Set message content type
SetContentType(ct ContentType) errors.SDKError
```

### Health Check

```go
// Service health check
Health(service string) (HealthInfo, errors.SDKError)
```

## Examples

### Domain and User Management

```go
ctx := context.Background()

// Create domain
domain := sdk.Domain{
    Name: "My Domain",
    Metadata: sdk.Metadata{"key": "value"},
}
domain, err := smqsdk.CreateDomain(ctx, domain, adminToken)

// Create user in domain
user := sdk.User{
    Name: "Jane Doe",
    Email: "jane@example.com",
    Credentials: sdk.Credentials{
        Username: "jane.doe",
        Secret:   "password123",
    },
}
user, err = smqsdk.CreateUser(ctx, user, adminToken)
```

### Client and Channel Operations

```go
// Create client
client := sdk.Client{
    Name: "Temperature Sensor",
    Metadata: sdk.Metadata{"location": "office"},
}
client, err := smqsdk.CreateClient(ctx, client, domainID, token)

// Create channel
channel := sdk.Channel{
    Name: "Temperature Data",
    Metadata: sdk.Metadata{"type": "sensor_data"},
}
channel, err = smqsdk.CreateChannel(ctx, channel, domainID, token)

// Connect client to channel
conn := sdk.Connection{
    ClientIDs:  []string{client.ID},
    ChannelIDs: []string{channel.ID},
    Types:      []string{"publish", "subscribe"},
}
err = smqsdk.Connect(ctx, conn, domainID, token)
```

### Group Management

```go
// Create group
group := sdk.Group{
    Name: "Sensors Group",
    Metadata: sdk.Metadata{"type": "sensors"},
}
group, err := smqsdk.CreateGroup(ctx, group, domainID, token)

// Set client parent group
err = smqsdk.SetClientParent(ctx, client.ID, domainID, group.ID, token)
```

### Role Management

```go
// Create domain role
roleReq := sdk.RoleReq{
    RoleName: "Editor",
    OptionalActions: []string{"read", "update"},
    OptionalMembers: []string{user.ID},
}
role, err := smqsdk.CreateDomainRole(ctx, domainID, roleReq, token)

// Add role members
members := []string{user.ID}
addedMembers, err := smqsdk.AddDomainRoleMembers(ctx, domainID, role.ID, members, token)
```
