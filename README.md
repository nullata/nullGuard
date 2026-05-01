# <img src="static/images/logo.png" alt="logo" width="24"> nullGuard

WireGuard VPN management application for creating and managing VPN servers and clients.

## Setup

### Requirements

The host or container must have:
- **wireguard-tools** installed (`wg`, `wg-quick`)
- **NET_ADMIN** capability (for managing WireGuard interfaces)
- **net.ipv4.ip_forward=1** sysctl (for routing VPN traffic)

### Database Setup

Create a MySQL/MariaDB database and user for nullguard:

```sql
CREATE DATABASE nullguard;
CREATE USER 'nullguard'@'%' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON nullguard.* TO 'nullguard'@'%';
FLUSH PRIVILEGES;
```

Tables are created automatically on first run.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NULLGUARD_PORT` | Port the app listens on | `8080` |
| `DB_HOST` | MySQL database host | _(required)_ |
| `DB_PORT` | MySQL database port | _(required)_ |
| `DB_NAME` | MySQL database name | _(required)_ |
| `DB_USER` | MySQL database user | _(required)_ |
| `DB_PASS` | MySQL database password | _(required)_ |
| `SESSION_SECRET_KEY` | Secret key for session encryption. Auto-generated if not set. | _(auto)_ |
| `WG_SERVER_CONF_PATH` | Path to store WireGuard config files | `./` |
| `AUTO_START_SERVERS` | Automatically start all WireGuard servers on application startup | `false` |
| `ENV` | Set to `production` for secure session cookies (HTTPS) | _(empty)_ |
| `WG_SERVER_DEFAULT_NAME` | Default name for new servers | _(empty)_ |
| `WG_SERVER_DEFAULT_ADDR` | Default address for new servers | _(empty)_ |
| `WG_SERVER_DEFAULT_PORT` | Default port for new servers | _(empty)_ |
| `WG_SERVER_DEFAULT_KEEPALIVE` | Default client keepalive for new servers (0-600 seconds) | _(empty)_ |
| `WG_SERVER_DEFAULT_COMMENT` | Default comment for new servers | _(empty)_ |
| `WG_SERVER_DEFAULT_SUPERNET` | Default supernet CIDR for new servers | _(empty)_ |
| `WG_CLIENT_DEFAULT_NAME` | Default name for new clients | _(empty)_ |

### Docker

nullGuard is available as a pre-built Docker image on [Docker Hub](https://hub.docker.com/r/nullata/nullguard):

```bash
docker pull nullata/nullguard
```

#### Docker Compose (Recommended)

Create a `docker-compose.yml` or use the one included in the repository. To use the pre-built image from Docker Hub, replace `build: .` with `image: nullata/nullguard:latest`:

```yaml
services:
  nullguard:
    image: nullata/nullguard:latest  # Use pre-built image
    # Or use: build: .  # Build locally
    ports:
      - "8080:8080"
      - "51820:51820/udp"
    # ... rest of configuration
```

Then run:

```bash
docker compose up -d
```

The app will be available at `http://localhost:8080`. On first visit you'll be prompted to set up an admin password.

**⚠️ Important:** Remember to configure port forwarding on your router to forward UDP port 51820 (or whichever ports you're using for WireGuard) to your Docker host's IP address. Without proper port forwarding, external clients won't be able to connect to your VPN.

### Web UI

nullGuard provides a web interface for managing WireGuard servers and clients:

- **Dashboard** - Overview of servers and clients
- **Server Management** - Create, update, delete, start/stop WireGuard servers
- **Client Management** - Create, update, delete clients with automatic configuration
- **QR Code Generation** - Scan QR codes to quickly set up mobile clients
- **Config Downloads** - Download client configuration files directly
- **Admin Settings** - Change password, manage API tokens, view password hints
- **Real-time Status** - Monitor server and client connection status

Access the web UI at `http://localhost:8080` after starting the application.

---

## Quick Start Example

1. Start the application and create an admin password at `http://localhost:8080/setup`
2. Create an API token from the web UI at **Admin Settings > API Token Management**
3. Create a WireGuard server (keys auto-generated):

```bash
curl -X POST http://localhost:8080/api/v1/create-server \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "interfaceName": "wg0",
    "address": "10.0.0.1/24",
    "port": 51820,
    "wanAddress": "vpn.example.com"
  }'
```

4. Start the server:

```bash
curl -X POST http://localhost:8080/api/v1/deploy-server \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "serverId": 1,
    "interfaceName": "wg0"
  }'
```

5. Create a client (keys and IP auto-assigned):

```bash
curl -X POST http://localhost:8080/api/v1/create-client \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "serverId": 1,
    "name": "my-laptop"
  }'
```

The response includes a ready-to-use WireGuard configuration in the `data.config` field.

6. Restart the server to apply the new client configuration:

```bash
curl -X POST http://localhost:8080/api/v1/restart-server \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "serverId": 1,
    "interfaceName": "wg0"
  }'
```

---

## How It Works

### Architecture

nullGuard manages WireGuard configurations through a server-client model:

- **Servers** represent WireGuard interfaces (e.g., `wg0`) with their own network, keys, and settings
- **Clients** are peers that connect to a server, each with their own keys and IP addresses
- All WireGuard keys and client IPs can be auto-generated if not provided
- Server configurations are stored in the database and generated as `.conf` files when deployed

### Key Concepts

**Auto-generation**: When creating servers or clients, you can omit keys, IPs, and other fields - the app will generate them automatically based on the server's configuration.

**Server state requirements**:
- Servers must be **stopped** before updating or deleting
- Servers must be **running** to restart them

**Client changes require server restart**: When you add, update, or delete a client, the server's WireGuard configuration file needs to be regenerated with the new peer list. Changes take effect only after restarting the server (via `restart-server` endpoint or the web UI).

**Authentication for destructive operations**: Endpoints that deploy, stop, restart, or delete servers require both `serverId` and the server's `interfaceName` for verification. This prevents accidental operations on the wrong server.

**Bridge networks**: Each server can declare a list of reachable server-side LANs (the `bridgeNetworks` field). When creating or editing a client, those CIDRs render as checkboxes in the web UI - ticking one appends it to that client's `AllowedIPs`, unticking removes it. Use this to grant individual clients access to LANs behind the server (e.g. a home subnet at `192.168.50.0/24`) without hand-editing the WireGuard config. Watch out for clients whose own local network overlaps a checked CIDR - that LAN gets routed through the VPN and the client loses local access while connected.

**LAN-to-LAN (peer-exposed networks)**: A client can declare CIDRs of LANs reachable *through itself* (the `exposedLans` field). On server-config generation those CIDRs are added to that client's `[Peer]` `AllowedIPs`, so the server routes inbound traffic for those subnets to that peer's tunnel. Other clients then see the exposed LANs as a "Reachable Peer LANs" checkbox group - ticking one appends the CIDR to that client's own `AllowedIPs` so it routes traffic for that subnet into the tunnel. The exposing client's host must have `net.ipv4.ip_forward=1` and either NAT/masquerade or LAN-side static routes configured - that is OS-level setup, not handled by nullguard. After any change to `exposedLans`, restart the server (or rely on auto-restart) so the new server config is loaded; other clients' downloadable `.conf` files also need to be re-fetched.

#### Setting up a LAN-to-LAN client on Ubuntu

Suppose a client host sits on `192.168.10.0/24` via its `eth0` interface and you want other VPN peers to reach that subnet. After setting `exposedLans: 192.168.10.0/24` in nullguard and restarting the server, do the following on that Ubuntu host:

**1. Enable IP forwarding**

```bash
# Apply immediately
sudo sysctl -w net.ipv4.ip_forward=1

# Persist across reboots
echo "net.ipv4.ip_forward=1" | sudo tee /etc/sysctl.d/99-ip-forward.conf
sudo sysctl -p /etc/sysctl.d/99-ip-forward.conf
```

**2. Masquerade outbound traffic toward the LAN**

This NATes the source IP of incoming VPN packets to the host's own LAN IP, so devices on `192.168.10.0/24` don't need a static route back to the VPN subnet.

```bash
# Replace eth0 with the interface facing your local LAN (check with: ip -br a)
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

# Persist the rule across reboots
sudo apt install -y iptables-persistent
sudo netfilter-persistent save
```

After these two steps, any VPN peer that has ticked `192.168.10.0/24` in its "Reachable Peer LANs" checkboxes (and re-downloaded its config) will be able to reach hosts on that subnet. The exposing client doesn't need to do anything special in WireGuard itself  the `AllowedIPs` expansion on the server side and the kernel forwarding + masquerade on the client side handle everything.

> **Note:** Masquerade hides the VPN peer's real address behind the exposing client's LAN IP. If you need the original VPN addresses visible on the LAN (e.g. for firewall rules), add a static route on your LAN gateway pointing the VPN subnet (`10.240.0.0/24`) via the exposing client's LAN IP instead of using masquerade.

---

## API

nullGuard exposes a REST API for programmatic management of servers and clients. All API endpoints require authentication via a Bearer token.

### Authentication

API tokens are created from the web UI at **Admin Settings > API Token Management**. Include the token in the `Authorization` header:

```
Authorization: Bearer <your-api-token>
```

All responses follow a standard format:

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Description",
  "data": null
}
```

### Endpoints

#### List Servers

Returns all WireGuard servers with basic details.

```
GET /api/v1/list-servers
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server list",
  "data": [
    {
      "id": 1,
      "interfaceName": "wg0",
      "address": "10.0.0.1/24"
    }
  ]
}
```

---

#### Create Server

Creates a new WireGuard server configuration.

```
POST /api/v1/create-server
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `interfaceName` | string | yes | Server interface name (e.g., "wg0") |
| `address` | string | yes | Server address in CIDR notation (e.g., "10.0.0.1/24") |
| `port` | number | yes | WireGuard listen port (e.g., 51820) |
| `publicKey` | string | no | Server public key. Auto-generated if omitted. |
| `privateKey` | string | no | Server private key. Auto-generated if omitted. |
| `postUp` | string | no | PostUp iptables rules |
| `postDown` | string | no | PostDown iptables rules |
| `wanAddress` | string | yes | Public IP or domain for clients to connect to |
| `supernetCidr` | string | no | Supernet CIDR for routing |
| `bridgeNetworks` | string | no | Comma-separated CIDRs for server-side LANs that clients may bridge into. Each entry becomes a checkbox on the client form (e.g. `"192.168.50.0/24, 10.10.0.0/16"`). |
| `defaultKeepAlive` | number | no | Default keepalive for clients (0-600 seconds, default: 30) |
| `comment` | string | no | Server description/comment |

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server created",
  "data": null
}
```

---

#### Fetch Server

Retrieves complete details for a specific server including status.

```
POST /api/v1/fetch-server
Content-Type: application/json
```

**Request body:**

```json
{
  "serverId": 1
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server data",
  "data": {
    "ID": 1,
    "InterfaceName": "wg0",
    "Comment": "Production VPN",
    "Address": "10.0.0.1/24",
    "Port": 51820,
    "PublicKey": "...",
    "PrivateKey": "...",
    "PostUp": "...",
    "PostDown": "...",
    "WANAddress": "vpn.example.com",
    "SupernetCidr": null,
    "DefaultKeepalive": 30,
    "AutoRestart": false,
    "IsActive": true
  }
}
```

---

#### Update Server

Updates an existing WireGuard server configuration. Server must be stopped before updating.

```
PUT /api/v1/update-server
Content-Type: application/json
```

**Request body:** Same fields as Create Server, plus `serverId` field.

> **Note:** Submitting an optional field as an empty string (or omitting it from the body) will **clear** the stored value. This applies to `postUp`, `postDown`, `supernetCidr`, `bridgeNetworks`, and `defaultKeepAlive`. Always include the fields you want preserved.

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server updated",
  "data": null
}
```

---

#### Delete Server

Deletes a WireGuard server and its configuration. Server must be stopped before deletion.

```
DELETE /api/v1/delete-server
Content-Type: application/json
```

**Request body:**

```json
{
  "serverId": 1,
  "interfaceName": "wg0"
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server configuration deleted",
  "data": null
}
```

---

#### Deploy (Start) Server

Starts a WireGuard server.

```
POST /api/v1/deploy-server
Content-Type: application/json
```

**Request body:**

```json
{
  "serverId": 1,
  "interfaceName": "wg0"
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server deployed and started: wg0",
  "data": null
}
```

---

#### Stop Server

Stops a running WireGuard server.

```
POST /api/v1/stop-server
Content-Type: application/json
```

**Request body:**

```json
{
  "serverId": 1,
  "interfaceName": "wg0"
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server stopped: wg0",
  "data": null
}
```

---

#### List Clients

Returns all clients for a given server with basic details.

```
GET /api/v1/list-clients/{serverId}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Client list",
  "data": [
    {
      "id": 1,
      "name": "my-laptop"
    },
    {
      "id": 2,
      "name": "my-phone"
    }
  ]
}
```

---

#### Create Client

Creates a new WireGuard client on a server.

```
POST /api/v1/create-client
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serverId` | number | yes | ID of the server to create the client on |
| `name` | string | yes | Client name (alphanumeric, dots, dashes, underscores) |
| `address` | string | no | Client IP address in CIDR notation. Auto-assigned if omitted. |
| `allowedIps` | string | no | Allowed IPs for the client. Derived from server subnet if omitted. |
| `dnsServers` | string | no | DNS servers for the client, comma-delimited (max 2). Omit for no DNS. Example: `"8.8.8.8, 1.1.1.1"` |
| `fullTunnel` | boolean | no | Route all traffic through the VPN (default: `false`) |
| `keepalive` | number | no | Persistent keepalive interval in seconds (0-600). Uses server default (30) if omitted. |
| `exposedLans` | string | no | Comma-separated CIDRs of LANs reachable through this client. Added to the server's route to this peer so other clients can reach them. Requires IP forwarding + masquerade on the client host. |

**Validation:**
- Keepalive must be between 0-600 seconds (WireGuard specification)
- Client names must be alphanumeric with dots, dashes, or underscores
- Addresses must be valid CIDR notation
- Each entry in `exposedLans` must be a valid CIDR

WireGuard keys, client address, and allowed IPs are all derived automatically from the server configuration when not provided.

**Minimal example:**

```json
{
  "serverId": 1,
  "name": "my-laptop"
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Client created",
  "data": {
    "config": "[Interface]\nPrivateKey = ...\nAddress = 10.0.0.2/32\n\n[Peer]\nPublicKey = ...\nEndpoint = vpn.example.com:51820\nAllowedIPs = 10.0.0.0/24\nPersistentKeepalive = 30\n"
  }
}
```

The `config` field contains the ready-to-use WireGuard client configuration file content.

---

#### Delete Client

Deletes a WireGuard client by client ID and server ID.

```
DELETE /api/v1/delete-client
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `clientId` | number | yes | ID of the client to delete |
| `serverId` | number | yes | ID of the server the client belongs to |

**Example:**

```json
{
  "clientId": 5,
  "serverId": 1
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Success",
  "data": null
}
```

---

#### Update Client

Updates an existing WireGuard client configuration. Changes are applied after the server is restarted.

```
PUT /api/v1/update-client
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `clientId` | number | yes | Client ID |
| `serverId` | number | yes | Server ID the client belongs to |
| `name` | string | yes | Client name |
| `address` | string | yes | Client IP address in CIDR notation |
| `allowedIps` | string | yes | Allowed IPs for the client |
| `dnsServers` | string | no | DNS servers, comma-delimited |
| `fullTunnel` | boolean | no | Route all traffic through VPN |
| `keepalive` | number | no | Keepalive interval (0-600 seconds) |
| `exposedLans` | string | no | Comma-separated CIDRs of LANs reachable through this client. Submit `""` to clear. |
| `publicKey` | string | yes | Client public key |
| `privateKey` | string | yes | Client private key |

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Client updated. Changes will be applied after the server is restarted",
  "data": null
}
```

---

#### Get Client Config

Retrieves the WireGuard configuration file content for a client.

```
GET /api/v1/client/{serverId}/{clientId}/config
```

**Response:**

Plain text WireGuard configuration:

```
[Interface]
PrivateKey = ...
Address = 10.0.0.2/32
DNS = 8.8.8.8, 1.1.1.1

[Peer]
PublicKey = ...
Endpoint = vpn.example.com:51820
AllowedIPs = 10.0.0.0/24
PersistentKeepalive = 30
```

---

#### Get Client QR Code

Generates a QR code image for the client configuration (useful for mobile devices).

```
GET /api/v1/client/{serverId}/{clientId}/qrcode
```

**Response:**

PNG image of QR code containing the client configuration.

---

#### Download Client Config

Downloads the client configuration as a `.conf` file.

```
GET /api/v1/client/{serverId}/{clientId}/download
```

**Response:**

File download with `Content-Disposition: attachment; filename="client-name.conf"`

---

#### Restart Server

Restarts a running WireGuard server (stop, regenerate config, start). The server must be currently active.

```
POST /api/v1/restart-server
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serverId` | number | yes | ID of the server to restart |
| `interfaceName` | string | yes | Interface name of the server |

**Example:**

```json
{
  "serverId": 1,
  "interfaceName": "wg0"
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Server restarted: wg0",
  "data": null
}
```

---

#### Toggle Auto-Restart

Enables or disables automatic server restart when clients are created, updated, or deleted. When enabled, the server will automatically regenerate its configuration and restart after any client modification (only if the server is currently active).

```
POST /api/v1/toggle-auto-restart
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serverId` | number | yes | ID of the server |
| `autoRestart` | boolean | yes | Enable or disable auto-restart |

**Example:**

```json
{
  "serverId": 1,
  "autoRestart": true
}
```

**Response:**

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Auto-restart updated",
  "data": null
}
```

---

### Error Responses

On failure, the `status` field will be `"error"` with a descriptive message:

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "error",
  "message": "Invalid data format",
  "data": null
}
```

Common HTTP status codes:
- `400` - Bad request (invalid input)
- `401` - Unauthorized (missing or invalid token)
- `405` - Method not allowed
- `500` - Internal server error

---

## Testing

Run the integration test suite against a live nullGuard instance:

```bash
cd integration-tests
./run.sh --host localhost --port 8080 --api-key <your-api-token>
```

**Options:**
- `--host` - Hostname or IP where nullGuard is running (default: `localhost`)
- `--port` - Port number (default: `8080`)
- `--api-key` - API token (required)
- `--server-name` - Custom server name for tests (default: `integration-test-<pid>`)

**Requirements:**
- `curl` and `jq` must be installed
- A running nullGuard instance with database access
- A valid API token

The test suite covers:
- Server CRUD operations (create, list, fetch, update, delete)
- Server lifecycle (deploy/start, restart, stop)
- Client CRUD operations (create, list, config/QR/download, delete)
- Bridge networks set/round-trip/clear and CIDR validation
- Client `exposedLans` accept/reject/clear
- Error cases (authentication, validation, not found)

See [integration-tests/run.sh](integration-tests/run.sh) for implementation details.

---

## Troubleshooting

### Common Issues

**"Could not find requested server" (HTTP 500)**

This error occurs when the `serverId` and `interfaceName` combination doesn't match a server in the database. Causes:
- Wrong `serverId`
- Wrong `interfaceName` (typo or case mismatch)
- Server was deleted

**Solution**: Verify the server ID and interface name using `GET /api/v1/list-servers` or `POST /api/v1/fetch-server`.

---

**"Server is active. Please stop the server before updating/deleting" (HTTP 500)**

Servers must be stopped before you can update or delete them.

**Solution**: Stop the server first using `POST /api/v1/stop-server`, then perform the update/delete operation.

---

**"Server is not currently active" (HTTP 400)**

The `restart-server` endpoint requires the server to be running.

**Solution**: If the server is stopped, use `deploy-server` instead to start it.

---

**Client changes not taking effect**

When you create, update, or delete a client, the changes are saved to the database but won't take effect until the server's WireGuard configuration is regenerated and the interface is restarted.

**Solution**: After modifying clients, restart the server using `POST /api/v1/restart-server`.

---

**"Public and Private keys cannot be empty" (HTTP 400)**

This occurs when creating/updating a server and the keys are missing or whitespace-only.

**Solution**: Either provide valid keys or omit them entirely to use auto-generation (for creation only).

---

## Development

### Building from Source

**Requirements:**
- Go 1.21 or later
- MySQL/MariaDB database
- WireGuard tools (`wg`, `wg-quick`)

**Build:**

```bash
git clone https://github.com/nullata/nullguard.git
cd nullguard
go build -o nullguard cmd/nullguard/main.go
```

**Run:**

```bash
./nullguard
```

Make sure to configure environment variables (see [Environment Variables](#environment-variables)) before running.

### Running Tests

Integration tests require a running instance:

```bash
# Start nullGuard
docker compose up -d

# Create an API token via the web UI
# Then run tests
cd integration-tests
./run.sh --host localhost --port 8080 --api-key <token>
```

---

## License

This project is licensed under the **Elastic License 2.0** - see the [LICENSE](LICENSE) file for details.

Copyright 2026 nullata

Included 3rd party licenses: [FontAwesome](static/fontawesome-free-7.1.0-web/LICENSE.txt)
