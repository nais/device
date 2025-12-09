# Deploy APIserver

Etter man har dyttet kode til repoet, og jobbene har kjørt kan du deploye ny apiserver.

1. `gcloud --project nais-device compute ssh --tunnel-through-iap apiserver`
2. `sudo -i`
3. `apt update`
4. `apt install apiserver`
