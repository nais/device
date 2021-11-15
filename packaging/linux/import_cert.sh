#!/usr/bin/env bash
set -e

download_cert() {
  curl --silent --fail https://outtune-api.prod-gcp.nais.io/local/cert --data @- << EOF | jq -r '.cert_pem' > cert.pem
  {
    "serial": "$(cat ~/.config/naisdevice/product_serial)",
    "public_key_pem": "$(base64 --wrap 0 <<< "$1")"
  }
EOF
}

main() {
  nss_databases=()
  if [[ -d "$HOME/.pki/nssdb" ]]; then
    nss_databases+=("sql:$HOME/.pki/nssdb")
  fi
  for ff_profile in "$HOME"/.mozilla/firefox/*.default-release*/; do
    nss_databases+=("$ff_profile")
  done
  echo "nss databases: ${nss_databases[*]}"

  if [[ ${#nss_databases[@]} -eq 0 ]]; then
    echo "no supported nss databases found."
    exit 1
  fi

  for db in "${nss_databases[@]}"; do
    echo "updating db: '$db'"
    # If key already enrolled:
    if certutil -d "$db" -L -n naisdevice &> /dev/null; then
      echo "renew cert"
      (
        set -e
        cd "$(mktemp -d)" && echo "working in: $(pwd)"

        pubkey="$(certutil -L -n naisdevice -d "$db" -a | openssl x509 -pubkey -noout -in -)"
        download_cert "$pubkey"
        certutil -d "$db" -D -n naisdevice
        certutil -d "$db" -A -n naisdevice -i cert.pem -t ,,

        rm -f cert.pem
        echo "import to '$db' done"
      )
    else
      echo "new cert"
      (
        set -e
        cd "$(mktemp -d)" && echo "working in: $(pwd)"

        openssl genrsa -out key.pem 4096
        pubkey="$(openssl rsa -in key.pem -pubout -outform PEM)"
        download_cert "$pubkey"
        openssl pkcs12 -export -out bundle.p12 -in cert.pem -inkey key.pem -password pass:asd123 -name naisdevice
        pk12util -d "$db" -i bundle.p12 -W asd123 -K ""

        rm -f key.pem cert.pem bundle.p12
        echo "import to '$db' done"
      )
    fi
  done
}

# update $db/ClientAuthRememberList.txt with cert prefs:
# nav-no.managed.us2.access-control.cas.ms:443
# nav-no.managed.prod04.access-control.cas.ms

# clean up old pubkey storage:
rm -f ~/.config/naisdevice/browser_cert_pubkey.pem

main
