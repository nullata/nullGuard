#!/usr/bin/env bash
#
# nullGuard API integration tests
# Runs sequential API tests against a live instance.
#
# Test sequence:
#   1. Health check  - verify the service is reachable
#   2. Server CRUD   - create a server (keys auto-generated), list servers,
#                      fetch server by id, update server comment, verify update
#   3. Server deploy - deploy (start) server, restart server, stop server
#   4. Client CRUD   - create a client (keys/address auto-generated), verify
#                      config is returned, list clients, get client config,
#                      get client QR code, download client config, delete
#                      client, verify client is removed
#   5. Cleanup       - delete the test server, verify it is removed from list
#   6. Error cases   - unauthenticated request (401), fetch non-existent
#                      server, create server with missing fields, create
#                      client without server id
#
# Usage:
#   ./run.sh --host <host> --port <port> --api-key <key> [--server-name <name>]
#
# Example:
#   ./run.sh --host localhost --port 8080 --api-key my-token
#   ./run.sh --host 192.168.1.50 --port 8080 --api-key my-token --server-name wg-test

set -euo pipefail

# check required dependencies
for cmd in curl jq; do
  if ! command -v "${cmd}" &>/dev/null; then
    echo "Error: '${cmd}' is required but not installed"
    exit 1
  fi
done

# defaults
host="localhost"
port="8080"
apiKey=""
serverName="integration-test-$$"  # unique per run

# parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)       host="$2";        shift 2 ;;
    --port)       port="$2";        shift 2 ;;
    --api-key)    apiKey="$2";     shift 2 ;;
    --server-name) serverName="$2"; shift 2 ;;
    -h|--help)
      sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "${apiKey}" ]]; then
  echo "Error: --api-key is required"
  exit 1
fi

baseUrl="http://${host}:${port}"
authHeader="Authorization: Bearer ${apiKey}"

# state (populated during tests)
serverId=""
clientId=""

# helpers
pass=0
fail=0
total=0

red()   { printf "\033[0;31m%s\033[0m" "$*"; }
green() { printf "\033[0;32m%s\033[0m" "$*"; }
bold()  { printf "\033[1m%s\033[0m" "$*"; }

# runTest <name> <method> <path> <expectedHttpCode> [body]
runTest() {
  local name="$1" method="$2" path="$3" expectedCode="$4" body="${5:-}"
  total=$((total + 1))

  local curlArgs=(
    -s -w "\n%{http_code}"
    -X "${method}"
    -H "${authHeader}"
    -H "Content-Type: application/json"
  )

  if [[ -n "${body}" ]]; then
    curlArgs+=(-d "${body}")
  fi

  local output
  output=$(curl "${curlArgs[@]}" "${baseUrl}${path}" 2>&1) || true

  local httpCode response
  httpCode=$(echo "${output}" | tail -n1)
  response=$(echo "${output}" | sed '$d')

  if [[ "${httpCode}" == "${expectedCode}" ]]; then
    pass=$((pass + 1))
    echo "  $(green "PASS") ${name} (HTTP ${httpCode})"
  else
    fail=$((fail + 1))
    echo "  $(red "FAIL") ${name} - expected HTTP ${expectedCode}, got ${httpCode}"
    echo "        Response: ${response}"
  fi

  # export response for callers to inspect
  lastResponse="${response}"
  lastHttpCode="${httpCode}"
}

# banner
echo ""
echo "$(bold "nullGuard API Integration Tests")"
echo "Target: ${baseUrl}"
echo "Server name: ${serverName}"
echo "-------------------------------------------"

###############################
#  1. Health check
###############################
echo ""
echo "$(bold "[Health]")"
runTest "Health check" GET "/api/v1/health" 200

###############################
#  2. Server CRUD
###############################
echo ""
echo "$(bold "[Server]")"

# create server (keys auto-generated)
runTest "Create server" POST "/api/v1/create-server" 200 \
  "{\"interfaceName\":\"${serverName}\",\"address\":\"10.99.99.1/24\",\"port\":51899,\"wanAddress\":\"127.0.0.1\"}"

# list servers - find ours
runTest "List servers" GET "/api/v1/list-servers" 200

serverId=$(echo "${lastResponse}" | jq -r ".data[] | select(.interfaceName == \"${serverName}\") | .id")

if [[ -z "${serverId}" ]]; then
  echo "  $(red "ABORT") Could not find created server in list"
  exit 1
fi
echo "  ... server id: ${serverId}"

# fetch server - get keys for subsequent operations
runTest "Fetch server" POST "/api/v1/fetch-server" 200 \
  "{\"serverId\":${serverId}}"

serverPublicKey=$(echo "${lastResponse}" | jq -r '.data.PublicKey')
serverPrivateKey=$(echo "${lastResponse}" | jq -r '.data.PrivateKey')

# update server (change comment + set bridge networks) - keys required for update
runTest "Update server" PUT "/api/v1/update-server" 200 \
  "{\"serverId\":${serverId},\"interfaceName\":\"${serverName}\",\"address\":\"10.99.99.1/24\",\"port\":51899,\"publicKey\":\"${serverPublicKey}\",\"privateKey\":\"${serverPrivateKey}\",\"wanAddress\":\"127.0.0.1\",\"comment\":\"updated by integration test\",\"bridgeNetworks\":\"192.168.50.0/24, 10.10.0.0/16\"}"

# fetch again to verify update
runTest "Verify server update" POST "/api/v1/fetch-server" 200 \
  "{\"serverId\":${serverId}}"

updatedComment=$(echo "${lastResponse}" | jq -r '.data.Comment')
total=$((total + 1))
if [[ "${updatedComment}" == "updated by integration test" ]]; then
  pass=$((pass + 1))
  echo "  $(green "PASS") Server comment updated correctly"
else
  fail=$((fail + 1))
  echo "  $(red "FAIL") Server comment mismatch: got '${updatedComment}'"
fi

# verify bridge networks round-trip
updatedBridges=$(echo "${lastResponse}" | jq -r '.data.BridgeNetworks // empty')
total=$((total + 1))
if [[ "${updatedBridges}" == "192.168.50.0/24, 10.10.0.0/16" ]]; then
  pass=$((pass + 1))
  echo "  $(green "PASS") Bridge networks set and round-tripped"
else
  fail=$((fail + 1))
  echo "  $(red "FAIL") Bridge networks mismatch: got '${updatedBridges}'"
fi

# reject invalid CIDRs in bridge networks
runTest "Reject invalid bridge CIDR" PUT "/api/v1/update-server" 400 \
  "{\"serverId\":${serverId},\"interfaceName\":\"${serverName}\",\"address\":\"10.99.99.1/24\",\"port\":51899,\"publicKey\":\"${serverPublicKey}\",\"privateKey\":\"${serverPrivateKey}\",\"wanAddress\":\"127.0.0.1\",\"bridgeNetworks\":\"not-a-cidr\"}"

# clear bridge networks (PUT with empty string) - exercises UpdatePointerFieldIfChanged nil-clear path
runTest "Clear bridge networks" PUT "/api/v1/update-server" 200 \
  "{\"serverId\":${serverId},\"interfaceName\":\"${serverName}\",\"address\":\"10.99.99.1/24\",\"port\":51899,\"publicKey\":\"${serverPublicKey}\",\"privateKey\":\"${serverPrivateKey}\",\"wanAddress\":\"127.0.0.1\",\"comment\":\"updated by integration test\",\"bridgeNetworks\":\"\"}"

runTest "Verify bridge networks cleared" POST "/api/v1/fetch-server" 200 \
  "{\"serverId\":${serverId}}"

clearedBridges=$(echo "${lastResponse}" | jq -r '.data.BridgeNetworks // "null"')
total=$((total + 1))
if [[ "${clearedBridges}" == "null" || -z "${clearedBridges}" ]]; then
  pass=$((pass + 1))
  echo "  $(green "PASS") Bridge networks cleared"
else
  fail=$((fail + 1))
  echo "  $(red "FAIL") Bridge networks not cleared: got '${clearedBridges}'"
fi

###############################
#  3. Server deploy
###############################
echo ""
echo "$(bold "[Server deploy]")"

deployBody="{\"serverId\":${serverId},\"interfaceName\":\"${serverName}\"}"

# deploy (start) server
runTest "Deploy server" POST "/api/v1/deploy-server" 200 "${deployBody}"

# restart server (must be running)
runTest "Restart server" POST "/api/v1/restart-server" 200 "${deployBody}"

# stop server
runTest "Stop server" POST "/api/v1/stop-server" 200 "${deployBody}"

###############################
#  4. Client CRUD
###############################
echo ""
echo "$(bold "[Client]")"

# create client (minimal - keys and address auto-generated)
runTest "Create client" POST "/api/v1/create-client" 200 \
  "{\"serverId\":${serverId},\"name\":\"test-client-$$\"}"

# verify config is returned
total=$((total + 1))
config=$(echo "${lastResponse}" | jq -r '.data.config // empty')
if [[ -n "${config}" ]]; then
  pass=$((pass + 1))
  echo "  $(green "PASS") Client config returned in response"
else
  fail=$((fail + 1))
  echo "  $(red "FAIL") Client config missing from create response"
fi

# list clients
runTest "List clients" GET "/api/v1/list-clients/${serverId}" 200

clientId=$(echo "${lastResponse}" | jq -r '.data[0].id // empty')

if [[ -z "${clientId}" ]]; then
  echo "  $(red "ABORT") Could not find created client in list"
  exit 1
fi
echo "  ... client id: ${clientId}"

# get client config
runTest "Get client config" GET "/api/v1/client/${serverId}/${clientId}/config" 200

# get client QR code (should return PNG image)
runTest "Get client QR code" GET "/api/v1/client/${serverId}/${clientId}/qrcode" 200

# download client config
runTest "Download client config" GET "/api/v1/client/${serverId}/${clientId}/download" 200

# delete client
runTest "Delete client" DELETE "/api/v1/delete-client" 200 \
  "{\"clientId\":${clientId},\"serverId\":${serverId}}"

# verify client is gone
runTest "List clients after delete" GET "/api/v1/list-clients/${serverId}" 200

clientCount=$(echo "${lastResponse}" | jq '.data | length')

total=$((total + 1))
if [[ "${clientCount}" == "0" ]]; then
  pass=$((pass + 1))
  echo "  $(green "PASS") Client list empty after delete"
else
  fail=$((fail + 1))
  echo "  $(red "FAIL") Expected 0 clients, got ${clientCount}"
fi

###############################
#  5. Cleanup - delete the test server
###############################
echo ""
echo "$(bold "[Cleanup]")"

runTest "Delete server" DELETE "/api/v1/delete-server" 200 \
  "{\"serverId\":${serverId},\"interfaceName\":\"${serverName}\"}"

# verify server is gone
runTest "List servers after delete" GET "/api/v1/list-servers" 200

foundAfter=$(echo "${lastResponse}" | jq "[.data[] | select(.interfaceName == \"${serverName}\")] | length")

total=$((total + 1))
if [[ "${foundAfter}" == "0" ]]; then
  pass=$((pass + 1))
  echo "  $(green "PASS") Server removed from list after delete"
else
  fail=$((fail + 1))
  echo "  $(red "FAIL") Server still present after delete"
fi

###############################
#  6. Error cases
###############################
echo ""
echo "$(bold "[Error cases]")"

# missing auth
total=$((total + 1))
errOutput=$(curl -s -w "\n%{http_code}" -X GET "${baseUrl}/api/v1/list-servers" 2>&1) || true
errCode=$(echo "${errOutput}" | tail -n1)
if [[ "${errCode}" == "401" ]]; then
  pass=$((pass + 1))
  echo "  $(green "PASS") Unauthenticated request returns 401"
else
  fail=$((fail + 1))
  echo "  $(red "FAIL") Unauthenticated request - expected 401, got ${errCode}"
fi

# invalid server id for fetch
runTest "Fetch non-existent server" POST "/api/v1/fetch-server" 500 \
  "{\"serverId\":999999}"

# create server with missing required fields
runTest "Create server missing fields" POST "/api/v1/create-server" 400 \
  "{\"interfaceName\":\"\"}"

# create client with missing server id
runTest "Create client missing serverId" POST "/api/v1/create-client" 400 \
  "{\"name\":\"orphan-client\"}"

# summary
echo ""
echo "-------------------------------------------"
if [[ ${fail} -eq 0 ]]; then
  echo "$(green "All ${total} tests passed.")"
else
  echo "$(red "${fail}/${total} tests failed.") (${pass} passed)"
fi
echo ""

exit ${fail}
