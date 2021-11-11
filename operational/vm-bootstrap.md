# add to terraform
1. add to `github.com/naisdevice-terraform/gateways/terraform.tfvars`
2. apply
3. [onprem]:
  a. `gcloud iam service-accounts keys create --iam-account=<account_email> key.json`
  b. `cat key.json | pbcopy && rm key.json`
  c. ssh to new gateway, paste key at: `/root/sa.json`

# install ansible-pull
`sudo apt update && sudo apt install ansible`

# add crontab entry
```
sudo contab -e
*/5 * * * * [ $(pgrep ansible-pull -c) -eq 0 ] && /usr/bin/ansible-pull --only-if-changed -U https://github.com/nais/device ansible/site.yml -i /root/ansible-inventory.yaml >> /var/log/naisdevice/ansible.log
```

# add ansible-inventory.yaml
sudo vi /root/ansible-inventory.yaml
Example:
```yaml
all:
  vars:
    gcp_project: nais-prod-020f
    tunnel_ip: 10.255.240.6
    name: nais-device-gw-k8s-prod
  children:
    gateways:
      hosts:
        nais-device-gw-k8s-prod:
```
