#!/bin/bash
nss_databases=()
if [[ -d "$HOME/.pki/nssdb" ]]; then
  nss_databases+=("sql:$HOME/.pki/nssdb/")
fi
for ff_profile in "$HOME"/.mozilla/firefox/*.default-release*/; do
  nss_databases+=("$ff_profile")
done

if [[ ${#nss_databases[@]} -eq 0 ]]; then
  echo "no supported nss databases found."
  exit 1
fi

for db in "${nss_databases[@]}"; do
  while certutil -d "$db" -D -n naisdevice &> /dev/null; do
    echo "removed naisdevice cert from '$db'"
    continue
  done

done

echo "Done resettings browser certs"
