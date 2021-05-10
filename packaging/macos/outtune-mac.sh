#!/bin/bash


ready=$(ifconfig | grep "utun69:")
serial=$(/usr/sbin/ioreg -c IOPlatformExpertDevice -d 2 | /usr/bin/awk -F\" '/IOPlatformSerialNumber/{print $(NF-1)}')
cn="naisdevice - $serial is out of tune"

## Tests if first run - and offer certificates if it is
if [ ! -f firstrun ];then

## Generate public key and call API to get  
  openssl genrsa -out key.pem 4096
  openssl rsa -in key.pem -pubout -outform PEM -out pub.pem
  
  pubkey="$(base64 < pub.pem)"

  curl -v --fail https://outtune-api.prod-gcp.nais.io/cert --data @- << EOF | /usr/local/bin/jq -r '.cert_pem' > cert.pem
    {
      "serial": "$serial",
      "public_key_pem": "$pubkey"
    }
EOF

## join returned cert and key as .p12 bundle and import in keychain - Delete .p12 when done

  openssl pkcs12 -export -out certificate.pfx -inkey key.pem -in cert.pem \
    -passout pass:"$serial"

  security import certificate.pfx -P "$serial" -A #/dev/null 2>&1

  rm certificate.pfx
  
  touch firstrun
  
  osascript -e 'display notification "Your certificate has been installed" with title "Outtune"'

else

## delete expired cert
 
  security delete-certificate -c "$cn"

## renew cert and import in keychain 

 pubkey="$(base64 < pub.pem)"
 
  curl -v https://outtune-api.prod-gcp.nais.io/cert --data @- << EOF | /usr/local/bin/jq -r '.cert_pem' > cert.pem
    {
      "serial": "$serial",
      "public_key_pem": "$pubkey"
    }
EOF

  security import cert.pem
  
  identity_cert=$(security find-certificate -c "$cn" -Z | grep "SHA-1 hash:")
  certhash=$(echo $identity_cert| cut -c13-53)

      ## knytter cert til riktig URL
  security set-identity-preference -Z $certhash -s "https://nav-no.managed.us2.access-control.cas.ms/aad_login"
  
  
  osascript -e 'display notification "Your certificate has been renewed" with title "Outtune"'

fi
exit 0

