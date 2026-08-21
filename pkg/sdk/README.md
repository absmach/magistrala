# Magistrala Go SDK

Go SDK, a Go driver for Magistrala HTTP API.

Provides comprehensive functionality for system administration (provisioning), messaging, user management, workspace management, groups, channels, clients, certificates and invitations.

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
        DevicesURL:     "http://localhost:9000",
        ChannelsURL:    "http://localhost:9001",
        WorkspacesURL:     "http://localhost:8189",
        HTTPAdapterURL: "http://localhost:8008",
        CertsURL:       "http://localhost:9019",
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
    DevicesURL      string
    UsersURL        string
    GroupsURL       string
    ChannelsURL     string
    WorkspacesURL      string
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

### Workspace Management

```go
// Create workspace
CreateWorkspace(ctx context.Context, d Workspace, token string) (Workspace, errors.SDKError)

// Get workspace information
Workspace(ctx context.Context, workspaceID, token string) (Workspace, errors.SDKError)

// List workspaces
Workspaces(ctx context.Context, pm PageMetadata, token string) (WorkspacesPage, errors.SDKError)

// Update workspace
UpdateWorkspace(ctx context.Context, d Workspace, token string) (Workspace, errors.SDKError)

// Workspace status management
EnableWorkspace(ctx context.Context, workspaceID, token string) errors.SDKError
DisableWorkspace(ctx context.Context, workspaceID, token string) errors.SDKError
FreezeWorkspace(ctx context.Context, workspaceID, token string) errors.SDKError

// Workspace roles management
CreateWorkspaceRole(ctx context.Context, id string, rq RoleReq, token string) (Role, errors.SDKError)
WorkspaceRoles(ctx context.Context, id string, pm PageMetadata, token string) (RolesPage, errors.SDKError)
WorkspaceRole(ctx context.Context, id, roleID, token string) (Role, errors.SDKError)
UpdateWorkspaceRole(ctx context.Context, id, roleID, newName string, token string) (Role, errors.SDKError)
DeleteWorkspaceRole(ctx context.Context, id, roleID, token string) errors.SDKError

// Workspace role actions management
AddWorkspaceRoleActions(ctx context.Context, id, roleID string, actions []string, token string) ([]string, errors.SDKError)
WorkspaceRoleActions(ctx context.Context, id, roleID string, token string) ([]string, errors.SDKError)
RemoveWorkspaceRoleActions(ctx context.Context, id, roleID string, actions []string, token string) errors.SDKError
RemoveAllWorkspaceRoleActions(ctx context.Context, id, roleID, token string) errors.SDKError
AvailableWorkspaceRoleActions(ctx context.Context, token string) ([]string, errors.SDKError)

// Workspace role members management
AddWorkspaceRoleMembers(ctx context.Context, id, roleID string, members []string, token string) ([]string, errors.SDKError)
WorkspaceRoleMembers(ctx context.Context, id, roleID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)
RemoveWorkspaceRoleMembers(ctx context.Context, id, roleID string, members []string, token string) errors.SDKError
RemoveAllWorkspaceRoleMembers(ctx context.Context, id, roleID, token string) errors.SDKError
ListWorkspaceMembers(ctx context.Context, workspaceID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Client Management

```go
// Create clients
CreateClient(ctx context.Context, client Client, workspaceID, token string) (Client, errors.SDKError)
CreateClients(ctx context.Context, client []Client, workspaceID, token string) ([]Client, errors.SDKError)

// Get client information
Client(ctx context.Context, id, workspaceID, token string) (Client, errors.SDKError)
Clients(ctx context.Context, pm PageMetadata, workspaceID, token string) (ClientsPage, errors.SDKError)

// Update clients
UpdateClient(ctx context.Context, client Client, workspaceID, token string) (Client, errors.SDKError)
UpdateClientTags(ctx context.Context, client Client, workspaceID, token string) (Client, errors.SDKError)
UpdateClientSecret(ctx context.Context, id, secret, workspaceID, token string) (Client, errors.SDKError)

// Client status management
EnableClient(ctx context.Context, id, workspaceID, token string) (Client, errors.SDKError)
DisableClient(ctx context.Context, id, workspaceID, token string) (Client, errors.SDKError)
DeleteClient(ctx context.Context, id, workspaceID, token string) errors.SDKError

// Client hierarchy management
SetClientParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError
RemoveClientParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError

// Client roles management
CreateClientRole(ctx context.Context, id, workspaceID string, rq RoleReq, token string) (Role, errors.SDKError)
ClientRoles(ctx context.Context, id, workspaceID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)
ClientRole(ctx context.Context, id, roleID, workspaceID, token string) (Role, errors.SDKError)
UpdateClientRole(ctx context.Context, id, roleID, newName, workspaceID string, token string) (Role, errors.SDKError)
DeleteClientRole(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError

// Client role actions management
AddClientRoleActions(ctx context.Context, id, roleID, workspaceID string, actions []string, token string) ([]string, errors.SDKError)
ClientRoleActions(ctx context.Context, id, roleID, workspaceID string, token string) ([]string, errors.SDKError)
RemoveClientRoleActions(ctx context.Context, id, roleID, workspaceID string, actions []string, token string) errors.SDKError
RemoveAllClientRoleActions(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError
AvailableClientRoleActions(ctx context.Context, workspaceID, token string) ([]string, errors.SDKError)

// Client role members management
AddClientRoleMembers(ctx context.Context, id, roleID, workspaceID string, members []string, token string) ([]string, errors.SDKError)
ClientRoleMembers(ctx context.Context, id, roleID, workspaceID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)
RemoveClientRoleMembers(ctx context.Context, id, roleID, workspaceID string, members []string, token string) errors.SDKError
RemoveAllClientRoleMembers(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError
ListClientMembers(ctx context.Context, clientID, workspaceID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Channel Management

```go
// Create channels
CreateChannel(ctx context.Context, channel Channel, workspaceID, token string) (Channel, errors.SDKError)
CreateChannels(ctx context.Context, channels []Channel, workspaceID, token string) ([]Channel, errors.SDKError)

// Get channel information
Channel(ctx context.Context, id, workspaceID, token string) (Channel, errors.SDKError)
Channels(ctx context.Context, pm PageMetadata, workspaceID, token string) (ChannelsPage, errors.SDKError)

// Update channels
UpdateChannel(ctx context.Context, channel Channel, workspaceID, token string) (Channel, errors.SDKError)
UpdateChannelTags(ctx context.Context, c Channel, workspaceID, token string) (Channel, errors.SDKError)

// Channel status management
EnableChannel(ctx context.Context, id, workspaceID, token string) (Channel, errors.SDKError)
DisableChannel(ctx context.Context, id, workspaceID, token string) (Channel, errors.SDKError)
DeleteChannel(ctx context.Context, id, workspaceID, token string) errors.SDKError

// Channel hierarchy management
SetChannelParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError
RemoveChannelParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError

// Channel connections
Connect(ctx context.Context, conn Connection, workspaceID, token string) errors.SDKError
Disconnect(ctx context.Context, conn Connection, workspaceID, token string) errors.SDKError
ConnectClients(ctx context.Context, channelID string, clientIDs, connTypes []string, workspaceID, token string) errors.SDKError
DisconnectClients(ctx context.Context, channelID string, clientIDs, connTypes []string, workspaceID, token string) errors.SDKError

// List channel members
ListChannelMembers(ctx context.Context, channelID, workspaceID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Group Management

```go
// Create group
CreateGroup(ctx context.Context, group Group, workspaceID, token string) (Group, errors.SDKError)

// Get group information
Group(ctx context.Context, id, workspaceID, token string) (Group, errors.SDKError)
Groups(ctx context.Context, pm PageMetadata, workspaceID, token string) (GroupsPage, errors.SDKError)

// Update groups
UpdateGroup(ctx context.Context, group Group, workspaceID, token string) (Group, errors.SDKError)
UpdateGroupTags(ctx context.Context, group Group, workspaceID, token string) (Group, errors.SDKError)

// Group status management
EnableGroup(ctx context.Context, id, workspaceID, token string) (Group, errors.SDKError)
DisableGroup(ctx context.Context, id, workspaceID, token string) (Group, errors.SDKError)
DeleteGroup(ctx context.Context, id, workspaceID, token string) errors.SDKError

// Group hierarchy management
SetGroupParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError
RemoveGroupParent(ctx context.Context, id, workspaceID, groupID, token string) errors.SDKError
AddChildren(ctx context.Context, id, workspaceID string, groupIDs []string, token string) errors.SDKError
RemoveChildren(ctx context.Context, id, workspaceID string, groupIDs []string, token string) errors.SDKError
RemoveAllChildren(ctx context.Context, id, workspaceID, token string) errors.SDKError
Children(ctx context.Context, id, workspaceID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError)
Hierarchy(ctx context.Context, id, workspaceID string, pm PageMetadata, token string) (GroupsHierarchyPage, errors.SDKError)

// Group roles management
CreateGroupRole(ctx context.Context, id, workspaceID string, rq RoleReq, token string) (Role, errors.SDKError)
GroupRoles(ctx context.Context, id, workspaceID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)
GroupRole(ctx context.Context, id, roleID, workspaceID, token string) (Role, errors.SDKError)
UpdateGroupRole(ctx context.Context, id, roleID, newName, workspaceID string, token string) (Role, errors.SDKError)
DeleteGroupRole(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError

// Group role actions management
AddGroupRoleActions(ctx context.Context, id, roleID, workspaceID string, actions []string, token string) ([]string, errors.SDKError)
GroupRoleActions(ctx context.Context, id, roleID, workspaceID string, token string) ([]string, errors.SDKError)
RemoveGroupRoleActions(ctx context.Context, id, roleID, workspaceID string, actions []string, token string) errors.SDKError
RemoveAllGroupRoleActions(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError
AvailableGroupRoleActions(ctx context.Context, id, token string) ([]string, errors.SDKError)

// Group role members management
AddGroupRoleMembers(ctx context.Context, id, roleID, workspaceID string, members []string, token string) ([]string, errors.SDKError)
GroupRoleMembers(ctx context.Context, id, roleID, workspaceID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)
RemoveGroupRoleMembers(ctx context.Context, id, roleID, workspaceID string, members []string, token string) errors.SDKError
RemoveAllGroupRoleMembers(ctx context.Context, id, roleID, workspaceID, token string) errors.SDKError
ListGroupMembers(ctx context.Context, groupID, workspaceID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)
```

### Certificate Management

```go
// Issue certificate for mTLS
IssueCert(ctx context.Context, clientID, validity, workspaceID, token string) (Cert, errors.SDKError)

// View certificate
ViewCert(ctx context.Context, certID, workspaceID, token string) (Cert, errors.SDKError)

// View certificates by client
ViewCertByClient(ctx context.Context, clientID, workspaceID, token string) (CertSerials, errors.SDKError)

// Revoke certificates
RevokeCert(ctx context.Context, certID, workspaceID, token string) (time.Time, errors.SDKError)
RevokeAllCerts(ctx context.Context, clientID, workspaceID, token string) (time.Time, errors.SDKError)
```

### Invitation Management

```go
// Send invitation
SendInvitation(ctx context.Context, invitation Invitation, token string) error

// List invitations
Invitations(ctx context.Context, pm PageMetadata, token string) (InvitationPage, error)
WorkspaceInvitations(ctx context.Context, pm PageMetadata, token, workspaceID string) (InvitationPage, error)

// Manage invitations
AcceptInvitation(ctx context.Context, workspaceID, token string) error
RejectInvitation(ctx context.Context, workspaceID, token string) error
DeleteInvitation(ctx context.Context, userID, workspaceID, token string) error
```

### Messaging

```go
// Send message to channel
SendMessage(ctx context.Context, workspaceID, topic, msg, secret string) errors.SDKError

// Set message content type
SetContentType(ct ContentType) errors.SDKError
```

### Health Check

```go
// Service health check
Health(service string) (HealthInfo, errors.SDKError)
```

## Examples

### Workspace and User Management

```go
ctx := context.Background()

// Create workspace
workspace := sdk.Workspace{
    Name: "My Workspace",
    Metadata: sdk.Metadata{"key": "value"},
}
workspace, err := smqsdk.CreateWorkspace(ctx, workspace, adminToken)

// Create user in workspace
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
client, err := smqsdk.CreateClient(ctx, client, workspaceID, token)

// Create channel
channel := sdk.Channel{
    Name: "Temperature Data",
    Metadata: sdk.Metadata{"type": "sensor_data"},
}
channel, err = smqsdk.CreateChannel(ctx, channel, workspaceID, token)

// Connect client to channel
conn := sdk.Connection{
    ClientIDs:  []string{client.ID},
    ChannelIDs: []string{channel.ID},
    Types:      []string{"publish", "subscribe"},
}
err = smqsdk.Connect(ctx, conn, workspaceID, token)
```

### Group Management

```go
// Create group
group := sdk.Group{
    Name: "Sensors Group",
    Metadata: sdk.Metadata{"type": "sensors"},
}
group, err := smqsdk.CreateGroup(ctx, group, workspaceID, token)

// Set client parent group
err = smqsdk.SetClientParent(ctx, client.ID, workspaceID, group.ID, token)
```

### Role Management

```go
// Create workspace role
roleReq := sdk.RoleReq{
    RoleName: "Editor",
    OptionalActions: []string{"read", "update"},
    OptionalMembers: []string{user.ID},
}
role, err := smqsdk.CreateWorkspaceRole(ctx, workspaceID, roleReq, token)

// Add role members
members := []string{user.ID}
addedMembers, err := smqsdk.AddWorkspaceRoleMembers(ctx, workspaceID, role.ID, members, token)
```
