# Komme seg inn på servere

## Enroll gateway:

1. Get admin token:

```
# in device repo
echo "export NAISDEVICE_ADMIN_PASSWORD=$(gcloud compute ssh --project nais-device --tunnel-through-iap apiserver -- sudo grep ADMIN /etc/default/apiserver|cut -d ':' -f 2)" > .env
source .env
go run ./cmd/controlplane-cli/ --apiserver 10.255.240.1:8099 gateway enroll --name <name> --endpoint '<public ip>:51820'
Follow cli instructions
```

## SSH til GCP noder (gateways, apiserver, prometheus...)

Du finner nodene i `nais-device` prosjektet.

`gcloud --project <project_id> compute ssh --tunnel-through-iap <hostname>`

## SSH til Azure gateways

Disse er satt opp som onprem gateways, bare bruk denne bstionen i steden for aura boksen:
`gcloud --project naisdevice compute ssh --tunnel-through-iap bastion`

## SSH til onprem gateways

1. Legg til ssh key under `admin_users` i [denne filen](/ansible/site.yml), push, vent til ansible har kjørt (cron hvert 5. minutt)
1. SSH til aura boksen (`ssh a01apvl099.adeo.no`), deretter `ssh <hostname eller public ip>` (finnes som kommentarer i [denne filen](https://github.com/nais/naisdevice-terraform/blob/master/terraform.tfvars))

## xpanes

### Nav gcp gateways

Må kjøres fra `naisdevice-terraform` repoet, med `terraform init` kjørt for å ha tilgang på state.

```
terraform state pull | jq -r '.resources[] | select(.type=="google_compute_instance" and .name=="gateway") | .instances[] | "gcloud compute ssh --tunnel-through-iap --project " + .attributes.project  + " " +.attributes.name' | xpanes -c '{}'
```

### Nav onprem gateways

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
xpanes --ssh a30drvl0{19..43}.oera.no a30drvl0{45..50}.oera.no
```

### Tenant management apiservere

```
for p in $(gcloud projects list | grep nais-management | cut -d ' ' -f 1); do
  echo gcloud compute ssh --tunnel-through-iap --project="$p" naisdevice-apiserver
done | xpanes -c '{}'
```

_Merk at Nav har sin Apiserver i et legacy prosjekt `nais-device`!_  
TL;DR `gcloud compute ssh --zone "europe-north1-a" "apiserver" --project "nais-device" --tunnel-through-iap` for Nav.

### Tenant gateways

```shell
for project in $(gcloud projects list --filter "labels.naiscluster:true and labels.tenant:<tenant>" --format 'value(projectId)'); do
    for vm in $(gcloud compute instances list --project $project --filter "labels.usage:naisdevice" --format 'value(name)'); do
        echo gcloud compute ssh --tunnel-through-iap --project $project $vm
    done
done | xpanes -c '{}'
```
