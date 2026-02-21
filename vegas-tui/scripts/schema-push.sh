#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/.."
SCHEMA_FILE="$PROJECT_ROOT/schema.sql"

if [ -z "${DATABASE_URL:-}" ] && [ -f "$PROJECT_ROOT/.env" ]; then
  echo "Loading .env file..."
  set -a
  source "$PROJECT_ROOT/.env"
  set +a
fi

if [ -z "${DATABASE_URL:-}" ]; then
  echo "ERROR: DATABASE_URL is not set"
  exit 1
fi

echo "Pushing schema to database..."

# Extract hostname from DATABASE_URL
HOST=$(echo "$DATABASE_URL" | sed -n 's|.*@\([^:/]*\).*|\1|p')

if [ -z "$HOST" ]; then
  echo "Could not extract hostname from DATABASE_URL, trying direct connection..."
  psql "$DATABASE_URL" <"$SCHEMA_FILE"
  echo "Done."
  exit 0
fi

echo "Resolving $HOST to IPv4 (bypassing IPv6)..."

# Try multiple methods to get IPv4
IPV4=""

# Method 1: getent (glibc, most Linux)
if [ -z "$IPV4" ]; then
  IPV4=$(getent ahostsv4 "$HOST" 2>/dev/null | awk '{print $1; exit}') || true
fi

# Method 2: dig (bind-tools / dnsutils)
if [ -z "$IPV4" ]; then
  IPV4=$(dig +short A "$HOST" 2>/dev/null | head -1) || true
fi

# Method 3: host command
if [ -z "$IPV4" ]; then
  IPV4=$(host -t A "$HOST" 2>/dev/null | awk '/has address/ {print $4; exit}') || true
fi

# Method 4: nslookup
if [ -z "$IPV4" ]; then
  IPV4=$(nslookup "$HOST" 2>/dev/null | awk '/^Address: / && !/127.0.0.1/ {print $2; exit}') || true
fi

# Method 5: python one-liner (last resort)
if [ -z "$IPV4" ]; then
  IPV4=$(python3 -c "import socket; print(socket.getaddrinfo('$HOST', 5432, socket.AF_INET)[0][4][0])" 2>/dev/null) || true
fi

if [ -z "$IPV4" ]; then
  echo "WARNING: Could not resolve IPv4 address for $HOST"
  echo "Trying direct connection (may fail on IPv6-only networks)..."
  psql "$DATABASE_URL" <"$SCHEMA_FILE"
else
  echo "Resolved $HOST -> $IPV4"
  # Replace hostname with IPv4 in the URL
  DATABASE_URL_IPV4="${DATABASE_URL//$HOST/$IPV4}"
  # Ensure sslmode=require is set (verify-full would fail against an IP)
  if [[ "$DATABASE_URL_IPV4" == *"sslmode="* ]]; then
    # Replace any existing sslmode with require
    DATABASE_URL_IPV4=$(echo "$DATABASE_URL_IPV4" | sed 's/sslmode=[a-z-]*/sslmode=require/')
  elif [[ "$DATABASE_URL_IPV4" == *"?"* ]]; then
    DATABASE_URL_IPV4="${DATABASE_URL_IPV4}&sslmode=require"
  else
    DATABASE_URL_IPV4="${DATABASE_URL_IPV4}?sslmode=require"
  fi
  psql "$DATABASE_URL_IPV4" <"$SCHEMA_FILE"
fi

echo "Done."
