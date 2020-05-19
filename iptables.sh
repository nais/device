#!/usr/bin/env bash

gateway_ip=155.55.64.69
destination_ip=155.55.182.22
destination_port=443
wireguard_interface=wg0
default_interface=ens160

# Snat from gw to svc:
iptables -t nat -A POSTROUTING -o "$default_interface" -p tcp --dport "$destination_port" -d "${destination_ip}/32" -j SNAT --to-source "$gateway_ip"

# Allow forward to svc:
iptables -A FORWARD -i "$wireguard_interface" -o "$default_interface" -p tcp --syn --dport "$destination_port" -d "${destination_ip}/32" -m conntrack --ctstate NEW -j ACCEPT

# Generic
iptables -A FORWARD -i "$wireguard_interface" -o "$default_interface" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A FORWARD -i "$default_interface" -o "$wireguard_interface" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
