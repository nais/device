#!/usr/bin/env bash
usage() {
    echo "$0 up your-tunnel-ip endpoint-ip endpoint-public-key device-name"
    echo "$0 down"
}

case "$1" in
  "up")
    [ $# -ne 5 ] && usage && exit 1
    dev="$5"
    ip="$2"
    network="192.168.2.0/24"
    gw_public="$4"
    gw_endpoint="${3}:51820"
    gw_ip="192.168.2.1/32"

    sudo mkdir -p /etc/wireguard

    if ! sudo test -f "/etc/wireguard/controlplane.conf"; then
      sudo test -f "/etc/wireguard/private.key" || wg genkey | sudo tee "/etc/wireguard/private.key" > /dev/null
      cat << EOF | sudo tee /etc/wireguard/controlplane.conf
[Interface]
PrivateKey = $(cat /etc/wireguard/private.key)

[Peer]
PublicKey = $gw_public
AllowedIPs = $gw_ip
Endpoint = $gw_endpoint
EOF
      sudo chmod 600 /etc/wireguard/controlplane.conf
    fi

    sudo wireguard-go "$dev"
    sudo wg setconf "$dev" /etc/wireguard/controlplane.conf
    sudo ifconfig "$dev" inet "${ip}/32" "$ip" alias
    sudo ifconfig "$dev" mtu 1380
    sudo ifconfig "$dev" up
    sudo route -q -n add -inet "$network" -interface "$dev"
    sudo wg
    echo -e "done! your public key:\n$(sudo cat /etc/wireguard/private.key | wg pubkey)"
    ;;

  "down")
    echo "shutting down all wireguard-go tunnels"
    sudo killall wireguard-go
    ;;

  *)
    usage
    ;;
esac
