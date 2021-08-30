#!/usr/bin/env bash
set -ex

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <gateway_name> <gcp_project>"
  exit 1
fi

if [[ ! -f "/root/sa.json" ]]; then
  echo "You need to place appropriate sa json at /root/sa.json before running this script."
fi

mkdir -p /var/log/naisdevice
chmod 755 /var/log/naisdevice

name="$1"
project="$2"
proxy=""
proxy_yaml=""
role="gateways"
if [[ $(hostname) =~ a30drvl ]]; then
  # onprem settings
  role="onprem_gateways"
  proxy="http://webproxy-internett.nav.no:8088"
  proxy_yaml="proxy_env:
      https_proxy: $proxy"
fi

# Install Ansible
apt update --yes
apt install ansible --yes

# Configure ansible inventory
cat <<EOF > /root/ansible-inventory.yaml
all:
  vars:
    name: $name
    gcp_project: $project
    $proxy_yaml
  children:
    $role:
      hosts:
        $(hostname):
EOF

# Set up cron for Ansible
if ! crontab -l 2>/dev/null | grep "ansible-pull"; then
 ( crontab -l 2>/dev/null; echo "*/5 * * * * [ \$(pgrep ansible-pull -c) -eq 0 ] && HTTPS_PROXY=$proxy /usr/bin/ansible-pull --only-if-changed -U https://github.com/nais/device ansible/site.yml -i /root/ansible-inventory.yaml >> /var/log/naisdevice/ansible.log") | crontab -
fi

