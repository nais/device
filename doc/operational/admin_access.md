# Komme seg inn på servere

## SSH til GCP noder (gateways, apiserver, prometheus...)

`gcloud --project <project_id> compute ssh --tunnel-through-iap <hostname>`

## SSH til onprem gateways

1. Legg til ssh key under `admin_users` i [denne filen](/ansible/site.yml), push, vent til ansible har kjørt (cron hvert 5. minutt)
1. SSH til aura boksen (`ssh a01apvl099.adeo.no`), deretter `ssh <hostname eller public ip>`

## SSH til onprem gateways via naisdevice

For at dette skal fungere må du være tilkoblet til gatewayen.

1. IP til hver gateway kan man se [her](https://grafana.nais.io/d/XnwquxkGz/naisdevice?viewPanel=16)
2. `ssh 10.255.24[0-9].*`

## Enroll gateway:

1. Get admin token:

```
# in device repo
echo "export NAISDEVICE_ADMIN_PASSWORD=$(gcloud compute ssh --project nais-device --tunnel-through-iap apiserver -- sudo grep ADMIN /etc/default/apiserver|cut -d ':' -f 2)" > .env
source .env
go run ./cmd/controlplane-cli/ --apiserver 10.255.240.1:8099 gateway enroll --name <name> --endpoint '<public ip>:51820'
Follow cli instructions
```

## xpanes

### NAV gcp gateways
må kjøres fra `naisdevice-terraform` repoet, med `terraform init` kjørt for å ha tilgang på state.
```
terraform state pull | jq -r '.resources[] | select(.type=="google_compute_instance" and .name=="gateway") | .instances[] | "gcloud compute ssh --tunnel-through-iap --project " + .attributes.project  + " " +.attributes.name' | xpanes -c '{}'
```

### NAV onprem gateways
Krever ssh config, at du har bruker på VMen ([legges til her](../ansible/site.yml#L30)), og JITA til naisvakt aktivert.
```
Host naisvakt
  User username
  ForwardAgent yes
  Hostname 10.255.241.187
  IdentityFile ~/.ssh/id_ed25519

Host a30drvl*.oera.no
  User username
  IdentityFile ~/.ssh/id_ed25519
  ProxyJump naisvakt
```
Deretter kan man koble til med:
```
xpanes --ssh a30drvl0{19..43}.oera.no
```

### Tenant management apiservere
```
for p in $(gcloud projects list | grep nais-management | cut -d ' ' -f 1); do
  echo gcloud compute ssh --tunnel-through-iap --project="$p" naisdevice-apiserver
done | xpanes -c '{}'
```

### Tenant gateways

```shell
for project in $(gcloud projects list --filter "labels.naiscluster:true" --format 'value(projectId)'); do
    for vm in $(gcloud compute instances list --project $project --filter "labels.usage:naisdevice" --format 'value(name)'); do
        echo gcloud compute ssh --tunnel-through-iap --project $project $vm
    done
done | xpanes -c '{}'
```
