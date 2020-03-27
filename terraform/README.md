Made bucket with:
`gsutil mb -l europe-north1 -p nais-device gs://nais-device-tfstate`

To apply changes:
`./apply.sh -var apiserver_tunnel_ip=<apiserver_tunnel_ip>`
