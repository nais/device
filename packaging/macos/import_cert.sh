#!/bin/bash
jq="/Applications/naisdevice.app/Contents/MacOS/jq"
curl="$(command -v curl)"
pubkey_path="$HOME/Library/Application Support/naisdevice/browser_cert_pubkey.pem"

if [[ ! -x $jq ]]; then
  echo "\`jq\` not found, this is bundled with naisdevice - check \`ls $jq\`"
  exit 1
fi

if [[ ! -x $curl ]]; then
  echo "\`curl\` not found, install it and try again"
  exit 1
fi

serial=$(/usr/sbin/ioreg -c IOPlatformExpertDevice -d 2 | /usr/bin/awk -F\" '/IOPlatformSerialNumber/{print $(NF-1)}')
cn="naisdevice - $serial is out of tune"

download_cert() {
  $curl --silent --fail https://outtune-api.prod-gcp.nais.io/cert --data @- << EOF | $jq -r '.cert_pem' > cert.pem
  {
    "serial": "$serial",
    "public_key_pem": "$(base64 < "$HOME/Library/Application Support/naisdevice/browser_cert_pubkey.pem")"
  }
EOF
}

if [ ! -f "$pubkey_path" ]; then
  (
    set -eo pipefail
    cd "$(mktemp -d)"
    openssl genrsa -out key.pem 4096
    openssl rsa -in key.pem -pubout -outform PEM > "$pubkey_path"
    download_cert

    ## join returned cert and key as .p12 bundle and import in keychain - Delete .p12 when done
    openssl pkcs12 -export -out certificate.pfx -inkey key.pem -in cert.pem -passout pass:"$serial"
    security import certificate.pfx -P "$serial" -A #/dev/null 2>&1
    ) || (rm -f "$pubkey_path"; echo "failed aquiring cert (first time run)"; exit 1)
else
  ( 
    set -eo pipefail
    cd "$(mktemp -d)"
    ## delete expired cert
    security delete-certificate -c "$cn"

    ## renew cert and import in keychain
    download_cert
    security import cert.pem
    identity_cert=$(security find-certificate -c "$cn" -Z | grep "SHA-1 hash:")
    certhash=$(echo "$identity_cert" | cut -c13-53)

    ## set identity preference to use this cert automaticlaly for specified domains
    security set-identity-preference -Z "$certhash" -s "https://nav-no.managed.us2.access-control.cas.ms/aad_login"
  ) || (echo "failed renewing cert"; exit 1)
fi
