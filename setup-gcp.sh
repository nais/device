gcloud beta compute --project=nais-device instances create gateway-1 --zone=europe-north1-a --machine-type=f1-micro --tags=wireguard-gateway --image=ubuntu-1804-bionic-v20200317 --image-project=ubuntu-os-cloud

gcloud compute --project=nais-device firewall-rules create allow-wireguard --direction=INGRESS --priority=1000 --network=default --action=ALLOW --rules=udp:51820 --source-ranges=0.0.0.0/0 --target-tags=wireguard-gateway # wireguard firewall

