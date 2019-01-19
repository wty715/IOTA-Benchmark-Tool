curl http://10.0.0.61:14265 \
  -X POST \
  -H 'Content-Type: application/json' \
  -H 'X-IOTA-API-Version: 1' \
  -d '{"command": "getNodeInfo"}'
