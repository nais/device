# install ansible-pull
`apt install ansible`

# add crontab entry
```
*/2 * * * * /usr/bin/ansible-pull --only-if-changed -U https://github.com/nais/device ansible/site.yml -i /root/ansible-inventory.yaml >> /var/log/naisdevice/ansible.log
```

# add ansible-inventory.yaml

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
