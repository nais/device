#!/usr/bin/env bash

devicecode_resp=$(curl -XPOST https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/oauth2/devicecode -H "Content-Type: application/x-www-form-urlencoded" -d 'resource=https%3A%2F%2Fgraph.windows.net&client_id=5d69cfe1-b300-4a1a-95ec-4752d07ccab1')

echo $devicecode_resp | jq .

devicecode=$(echo $devicecode_resp | jq -r .device_code)

while sleep 5; do curl -d "grant_type=device_code&resource=https%3A%2F%2Fgraph.windows.net&code=${devicecode}&client_id=5d69cfe1-b300-4a1a-95ec-4752d07ccab1" https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/oauth2/token; done





	
