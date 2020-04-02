#!/usr/bin/env bash
usage() {
    echo "$0 up ctrl-tunnel-ip data-tunnel-ip"
    echo "$0 down"
}

case "$1" in
  "up")
    [ $# -ne 3 ] && usage && exit 1
    apiserver_endpoint="35.228.142.96:51820"
    apiserver_public_key="FUwVtyvs8nIRx9RpUUEopkfV8idmHz9g9K/vf9MFOXI="
    apiserver_tunnel_ip="10.255.240.1/32"
    wgctrl_device="utun8"
    wgctrl_tunnel_ip="$2"

    gateway_1_endpoint="35.228.118.232:51820"
    gateway_1_public_key="55h6JA2ZMPzaoa+iZU62JmqmtgK3ydj4YdT9HkkhnEQ="
    gateway_1_tunnel_ip="10.255.248.2/32"
    wgdata_device="utun9"
    wgdata_tunnel_ip="$3"

    sudo mkdir -p /etc/wireguard

    sudo test -f "/etc/wireguard/wgctrl-private.key" || wg genkey | sudo tee "/etc/wireguard/wgctrl-private.key" > /dev/null
    sudo test -f "/etc/wireguard/wgdata-private.key" || wg genkey | sudo tee "/etc/wireguard/wgdata-private.key" > /dev/null

    cat << EOF | sudo tee /etc/wireguard/wgctrl.conf
[Interface]
PrivateKey = $(cat /etc/wireguard/wgctrl-private.key)

[Peer]
PublicKey = $apiserver_public_key
AllowedIPs = $apiserver_tunnel_ip
Endpoint = $apiserver_endpoint
EOF

    cat << EOF | sudo tee /etc/wireguard/wgdata.conf
[Interface]
PrivateKey = $(cat /etc/wireguard/wgdata-private.key)

[Peer]
PublicKey = $gateway_1_public_key
AllowedIPs = $gateway_1_tunnel_ip
Endpoint = $gateway_1_endpoint
EOF

    sudo chmod 600 /etc/wireguard/wgctrl.conf
    sudo chmod 600 /etc/wireguard/wgdata.conf

    sudo wireguard-go "$wgctrl_device"
    sudo wg setconf "$wgctrl_device" /etc/wireguard/wgctrl.conf

    sudo ifconfig "$wgctrl_device" inet "${wgctrl_tunnel_ip}/21" "${wgctrl_tunnel_ip}" add
    sudo ifconfig "$wgctrl_device" mtu 1380
    sudo ifconfig "$wgctrl_device" up
    sudo route -q -n add -inet "${wgctrl_tunnel_ip}/21" -interface "$wgctrl_device"

    sudo wireguard-go "$wgdata_device"
    sudo wg setconf "$wgdata_device" /etc/wireguard/wgdata.conf

    sudo ifconfig "$wgdata_device" inet "${wgdata_tunnel_ip}/21" "${wgdata_tunnel_ip}" add
    sudo ifconfig "$wgdata_device" mtu 1380
    sudo ifconfig "$wgdata_device" up
    sudo route -q -n add -inet "${wgdata_tunnel_ip}/21" -interface "$wgdata_device"

    sudo wg

    ;;

  "down")
    echo "shutting down all wireguard-go tunnels"
    sudo killall wireguard-go
    ;;

  *)
    usage
    ;;
esac
