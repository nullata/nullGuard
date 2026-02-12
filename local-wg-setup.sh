#!/bin/bash

# fresh server install steps for nullguard - debian/ubuntu

apt install wireguard wireguard-tools -y

ufw allow 51820/udp

echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
sysctl -p
