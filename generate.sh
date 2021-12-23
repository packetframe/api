#!/usr/bin/bash

echo "# Packetframe API

### Routes:" > README.md
DOCUMENT=true go run cmd/api/main.go >> README.md
