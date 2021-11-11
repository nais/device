# Komme seg inn på servere

## SSH til GCP noder (gateways, apiserver, prometheus, bootstrap-api...)
`gcloud --project <project_id> compute ssh --tunnel-through-iap <hostname>`

## SSH til onprem gateways:
1. Legg til ssh key under `admin_users` i [denne filen](/ansible/site.yml), push, vent ~5 minutter
2. SSH til aura boksen (`ssh a01apvl099.adeo.no`), deretter `ssh <hostname eller public ip>`

## SSH til onprem gateways via naisdevice:
0. For at dette skal fungere må du være tilkoblet til gatewayen.
1. IP til hver gateway kan man se [her](https://grafana.nais.io/d/XnwquxkGz/naisdevice?viewPanel=16)
2. `ssh 10.255.24[0-9].*`
