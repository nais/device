#!/usr/bin/env bash

token_url="https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/oauth2/v2.0/token"
devicecode_url="https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/oauth2/v2.0/devicecode"
client_id="5d69cfe1-b300-4a1a-95ec-4752d07ccab1"
form_header="Content-Type: application/x-www-form-urlencoded"

devicecode_resp=$(curl -s -H "$form_header" -d "scope=$client_id/.default openid&client_id=$client_id" "$devicecode_url")

# jq . <<< "$devicecode_resp"

devicecode=$(jq -r .device_code <<< "$devicecode_resp")
uri=$(jq -r '.verification_uri' <<< "$devicecode_resp")
code=$(jq -r .user_code <<< "$devicecode_resp")

open "$uri"
pbcopy <<< "$code"

echo "Url: $uri, User code: $code"
echo "We opened the browser for you and copied the code to the clipboard 😘"
echo -n "Waiting for auth to complete "

while true; do
  token_resp=$(curl -s -H "$form_header" -d "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=${devicecode}&client_id=$client_id" "$token_url")

  if [[ $(jq -r .error <<< "$token_resp") == 'authorization_pending' ]]; then
    echo -n "."
    sleep 2
    continue
  else
    jq -r . <<< "$token_resp"

    id_token=$(jq -r '.id_token' <<< "$token_resp")
    access_token=$(jq -r '.access_token' <<< "$token_resp")

    jwt "$id_token"
    jwt "$access_token"
    break
  fi
done
