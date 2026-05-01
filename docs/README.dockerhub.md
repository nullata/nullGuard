<p align="center">
<img src="https://raw.githubusercontent.com/nullata/nullguard/main/static/images/logo.png" alt="Logo" width="96">
</p>

# nullGuard

WireGuard VPN management application for creating and managing VPN servers and clients through a web UI and REST API.

## Quick Start

Pull the image and start with Docker Compose:

```bash
docker pull nullata/nullguard:latest
```

### Docker Compose (Recommended)

Create a `docker-compose.yml`:

```yaml
services:
  nullguard:
    build: .
    ports:
      - "8080:8080"
      # WireGuard UDP ports (add udp ports for each WireGuard server instance)
      - "51820:51820/udp"
    environment:
      SERVER_PORT: "8080"
      SERVER_SSL_ENABLED: false
      DB_HOST: localhost
      DB_PORT: 3306
      DB_NAME: nullguard
      DB_USER: nullguard
      DB_PASS: changeme
      # Optional values
      SESSION_SECRET_KEY: ""
      WG_SERVER_DEFAULT_NAME: wg0
      WG_SERVER_DEFAULT_COMMENT: nullguard
      WG_SERVER_DEFAULT_ADDR: 10.252.0.1/24
      WG_SERVER_DEFAULT_PORT: 51820
      WG_SERVER_DEFAULT_SUPERNET: 128.0.0.0/1, 0.0.0.0/1
      WG_SERVER_DEFAULT_KEEPALIVE: 30
      WG_SERVER_DEFAULT_CONF_PATH: /etc/wireguard
      WG_SERVER_CONF_PATH: /etc/wireguard
      WG_CLIENT_DEFAULT_NAME: my-wg-client
      AUTO_START_SERVERS: false
      COOKIE_SECURE: false
      SESSION_MAX_AGE: 3600
      ENV: production
    restart: unless-stopped
    cap_add:
      - NET_ADMIN
    sysctls:
      - net.ipv4.ip_forward=1
```

Then run:

```bash
docker compose up -d
```

Access the web UI at `http://localhost:8080`. On first visit, you'll be prompted to set up an admin password.

**⚠️ Important:** Remember to configure port forwarding on your router to forward UDP port 51820 (or whichever ports you're using for WireGuard) to your Docker host's IP address. Without proper port forwarding, external clients won't be able to connect to your VPN.

## Requirements

The container requires:

- **NET_ADMIN** capability (for managing WireGuard interfaces)
- **net.ipv4.ip_forward=1** sysctl (for routing VPN traffic)
- **wireguard-tools** (included in the image)
- **MySQL/MariaDB** database

### Database Setup

Create a MySQL/MariaDB database and user:

```sql
CREATE DATABASE nullguard;
CREATE USER 'nullguard'@'%' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON nullguard.* TO 'nullguard'@'%';
FLUSH PRIVILEGES;
```

Tables are created automatically on first run.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NULLGUARD_PORT` | Port the app listens on | `8080` |
| `DB_HOST` | MySQL database host | _(required)_ |
| `DB_PORT` | MySQL database port | _(required)_ |
| `DB_NAME` | MySQL database name | _(required)_ |
| `DB_USER` | MySQL database user | _(required)_ |
| `DB_PASS` | MySQL database password | _(required)_ |
| `SESSION_SECRET_KEY` | Secret key for session encryption | _(auto-generated)_ |
| `WG_SERVER_CONF_PATH` | Path to store WireGuard config files | `./` |
| `AUTO_START_SERVERS` | Auto-start all WireGuard servers on startup | `false` |
| `ENV` | Set to `production` for secure session cookies | _(empty)_ |

**Optional defaults for new servers:**
- `WG_SERVER_DEFAULT_NAME` - Default server name
- `WG_SERVER_DEFAULT_ADDR` - Default server address
- `WG_SERVER_DEFAULT_PORT` - Default server port
- `WG_SERVER_DEFAULT_KEEPALIVE` - Default keepalive (0-600 seconds)
- `WG_SERVER_DEFAULT_SUPERNET` - Default supernet CIDR
- `WG_SERVER_DEFAULT_COMMENT` - Default server comment

**Optional defaults for new clients:**
- `WG_CLIENT_DEFAULT_NAME` - Default client name

## Features

- **Web UI** - Dashboard, server/client management, QR codes, config downloads
- **REST API** - Full programmatic control of servers and clients
- **Auto-generation** - WireGuard keys and client IPs generated automatically
- **Real-time Status** - Monitor server and client connection status
- **Easy Config** - Download client configs or scan QR codes for mobile devices

## API Quick Start

1. Create an admin password at `http://localhost:8080/setup`
2. Generate an API token from **Admin Settings > API Token Management**
3. Create a WireGuard server:

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

5. Create a client:

```bash
curl -X POST http://localhost:8080/api/v1/create-client \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "serverId": 1,
    "name": "my-laptop"
  }'
```

6. Restart the server to apply client changes:

```bash
curl -X POST http://localhost:8080/api/v1/restart-server \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "serverId": 1,
    "interfaceName": "wg0"
  }'
```

## API Documentation

### Authentication

All API endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer <your-api-token>
```

All responses follow this format:

```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "status": "success",
  "message": "Description",
  "data": null
}
```

### Server Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/list-servers` | List all servers |
| POST | `/api/v1/create-server` | Create a new server |
| POST | `/api/v1/fetch-server` | Get server details |
| PUT | `/api/v1/update-server` | Update a server (must be stopped) |
| DELETE | `/api/v1/delete-server` | Delete a server (must be stopped) |
| POST | `/api/v1/deploy-server` | Start a server |
| POST | `/api/v1/stop-server` | Stop a server |
| POST | `/api/v1/restart-server` | Restart a running server |

### Client Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/list-clients/{serverId}` | List all clients for a server |
| POST | `/api/v1/create-client` | Create a new client |
| PUT | `/api/v1/update-client` | Update a client |
| DELETE | `/api/v1/delete-client` | Delete a client |
| GET | `/api/v1/client/{serverId}/{clientId}/config` | Get client config file |
| GET | `/api/v1/client/{serverId}/{clientId}/qrcode` | Get client QR code (PNG) |
| GET | `/api/v1/client/{serverId}/{clientId}/download` | Download client config |

### Create Server

**POST** `/api/v1/create-server`

```json
{
  "interfaceName": "wg0",
  "address": "10.0.0.1/24",
  "port": 51820,
  "wanAddress": "vpn.example.com",
  "publicKey": "",         // Auto-generated if omitted
  "privateKey": "",        // Auto-generated if omitted
  "postUp": "",            // Optional iptables rules
  "postDown": "",          // Optional iptables rules
  "supernetCidr": "",      // Optional supernet CIDR
  "bridgeNetworks": "",    // Optional comma-separated CIDRs of server-side LANs clients may bridge into
  "defaultKeepAlive": 30,  // 0-600 seconds
  "comment": ""            // Optional description
}
```

### Create Client

**POST** `/api/v1/create-client`

```json
{
  "serverId": 1,
  "name": "my-laptop",
  "address": "",           // Auto-assigned if omitted
  "allowedIps": "",        // Derived from server if omitted
  "dnsServers": "8.8.8.8, 1.1.1.1",  // Optional, max 2
  "fullTunnel": false,     // Route all traffic through VPN
  "keepalive": 30,         // 0-600 seconds, uses server default if omitted
  "exposedLans": ""        // Optional comma-separated CIDRs of LANs reachable through this client (LAN-to-LAN)
}
```

**Response includes ready-to-use config:**

```json
{
  "status": "success",
  "message": "Client created",
  "data": {
    "config": "[Interface]\nPrivateKey = ...\n..."
  }
}
```

### Important Notes

- **Client changes require server restart** - After creating, updating, or deleting clients, restart the server for changes to take effect
- **Servers must be stopped** before updating or deleting
- **Auto-generation** - WireGuard keys, IPs, and allowed IPs are auto-generated when not provided
- **Interface name verification** - Destructive operations require both `serverId` and `interfaceName` to prevent accidents
- **Interface name length** - Linux limits interface names to 15 characters
- **Update-server clears omitted fields** - On `update-server`, omitting an optional field (or sending it as `""`) **clears** the stored value. Applies to `postUp`, `postDown`, `supernetCidr`, `bridgeNetworks`, and `defaultKeepAlive` - always include the fields you want preserved
- **Bridge networks** - A server's `bridgeNetworks` lists server-side LANs that the web UI surfaces as checkboxes on each client form; ticking one appends the CIDR to that client's `AllowedIPs`
- **LAN-to-LAN (`exposedLans`)** - A client may declare CIDRs reachable through itself; those CIDRs are added to that peer's server-side `AllowedIPs` so other clients can route to them. Requires `net.ipv4.ip_forward=1` and NAT/masquerade (or LAN-side static routes) on the exposing client's host - see the full README for the Ubuntu setup walkthrough

## Common Issues

**"Server is active. Please stop the server before updating/deleting"**
- Servers must be stopped before updating or deleting. Use `/api/v1/stop-server` first.

**"Server is not currently active"**
- The `restart-server` endpoint requires a running server. Use `deploy-server` to start it.

**Client changes not taking effect**
- Client modifications require a server restart. Use `/api/v1/restart-server`.

## Links

- **GitHub**: [github.com/nullata/nullguard](https://github.com/nullata/nullguard)
- **Full Documentation**: [github.com/nullata/nullguard/blob/main/README.md](https://github.com/nullata/nullguard/blob/main/README.md)
- **Report Issues**: [github.com/nullata/nullGuard/issues](https://github.com/nullata/nullGuard/issues)
- **Support the project:** [ko-fi.com/nickscripts](https://ko-fi.com/nickscripts)
- **Website:** [nickscripts.com](https://nickscripts.com)

## License

Licensed under the **Elastic License 2.0**

Copyright 2026 nullata
