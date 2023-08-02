#!/usr/bin/env bash
set -e

if [[ $# -ne 2 ]]; then
	echo "Usage: $0 <gateway_name> <gcp_project>"
	exit 1
fi

name="$1"
project="$2"
proxy_yaml=""
role="gateways"
# onprem settings
if [[ $(hostname) =~ a30drvl ]]; then
	if [[ ! -f "/root/sa.json" ]]; then
		echo "You need to place appropriate sa json at /root/sa.json before running this script."
		exit 1
	fi

	role="onprem_gateways"
	proxy_yaml="proxy_env:
      https_proxy: http://webproxy-internett.nav.no:8088"
	export HTTPS_PROXY="http://webproxy-internett.nav.no:8088"
fi

apt-get install --yes ca-certificates curl apt-transport-https gnupg

curl -L https://europe-north1-apt.pkg.dev/doc/repo-signing-key.gpg | gpg --dearmor >/etc/apt/trusted.gpg.d/nais-ppa-google-artifact-registry.gpg
echo 'deb [arch=amd64] https://europe-north1-apt.pkg.dev/projects/naisdevice controlplane main' >/etc/apt/sources.list.d/europe_north1_apt_pkg_dev_projects_naisdevice.list

apt update --yes
apt install ansible gateway-agent --yes

cat <<EOF >/root/ansible-inventory.yaml
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

echo "add the following line to crontab:"
echo "*/15 * * * * [ \$(pgrep ansible-pull -c) -eq 0 ] && HTTPS_PROXY=$HTTPS_PROXY /usr/bin/ansible-pull --only-if-changed -U https://github.com/nais/device ansible/site.yml -i /root/ansible-inventory.yaml >> /var/log/naisdevice/ansible.log"
