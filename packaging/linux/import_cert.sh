#!/usr/bin/env bash
set -e

download_cert() {
  curl --silent --fail https://outtune-api.prod-gcp.nais.io/cert --data @- << EOF | jq -r '.cert_pem' > cert.pem
  {
    "serial": "$(cat ~/.config/naisdevice/product_serial)",
    "public_key_pem": "$(base64 --wrap 0 < ~/.config/naisdevice/browser_cert_pubkey.pem)"
  }
EOF
}

main() {
  for profile in "sql:$HOME/.pki/nssdb" "$HOME"/.mozilla/firefox/*.default-release/; do
    echo "updating profile: $profile"
    # If key already enrolled:
    if certutil -d "$profile" -K -n naisdevice &> /dev/null; then
      echo "cert only import"
      (
        set -e
        cd "$(mktemp -d)" && echo "working in: $(pwd)"
        download_cert

        if certutil -d "$profile" -D -n naisdevice > /dev/null; then
          echo "removed old cert"
        else
          echo "failed to remove old cert or no old cert found"
        fi

        certutil -d "$profile" -A -n naisdevice -i cert.pem -t ,,
        rm -f cert.pem
        echo "done"
      )
    else
      echo "first time import"
      (
        set -e
        cd "$(mktemp -d)" && echo "working in: $(pwd)"
        openssl genrsa -out key.pem 4096
        openssl rsa -in key.pem -pubout -outform PEM > ~/.config/naisdevice/browser_cert_pubkey.pem
        download_cert

        openssl pkcs12 -export -out bundle.p12 -in cert.pem -inkey key.pem -password pass:asd123 -name naisdevice
        pk12util -d "$profile" -i bundle.p12 -W asd123

        rm -f key.pem cert.pem bundle.p12
        echo "done"
      )
    fi
  done
}

# update $profile/ClientAuthRememberList.txt with cert prefs:
# nav-no.managed.us2.access-control.cas.ms:443
# nav-no.managed.prod04.access-control.cas.ms

main
