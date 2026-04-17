#!/usr/bin/env bash
# 在 Mac 宿主机上以 sudo 执行：将 Fusion vmnet8（NAT）改为 192.168.56.0/24，
# 与 Ubuntu 静态 192.168.56.11 对齐。
#
# 执行前：关闭所有虚拟机与 VMware Fusion；已备份可回滚。
# 执行后：重新打开 VMware Fusion，再测：ping 192.168.56.11
#
# 用法：sudo bash /path/to/vmware-fusion-nat-to-192.168.56.sh

set -euo pipefail
PF="/Library/Preferences/VMware Fusion"
[[ -d "$PF" ]] || { echo "未找到: $PF"; exit 1; }

TS="$(date +%Y%m%d%H%M%S)"
cp -a "$PF/networking" "$PF/networking.bak.$TS"
cp -a "$PF/vmnet8/nat.conf" "$PF/vmnet8/nat.conf.bak.$TS"
cp -a "$PF/vmnet8/dhcpd.conf" "$PF/vmnet8/dhcpd.conf.bak.$TS"

sed -i '' 's/^answer VNET_8_HOSTONLY_SUBNET .*/answer VNET_8_HOSTONLY_SUBNET 192.168.56.0/' "$PF/networking"

cat > "$PF/vmnet8/nat.conf" << 'EOF'
# VMware NAT — 192.168.56.0/24（与 Ubuntu 192.168.56.11 同网段）
[host]
useMacosVmnetVirtApi = 1
ip = 192.168.56.2
netmask = 255.255.255.0
device = vmnet8
activeFTP = 1
allowAnyOUI = 1
resetConnectionOnLinkDown = 1
resetConnectionOnDestLocalHost = 1
natIp6Enable = 0
natIp6Prefix = fd15:4ba5:5a2b:1008::/64
hostIp = 192.168.56.1

[tcp]
timeWaitTimeout = 30

[udp]
timeout = 60

[netbios]
nbnsTimeout = 2
nbnsRetries = 3
nbdsTimeout = 3

[incomingtcp]
[incomingudp]
EOF

cat > "$PF/vmnet8/dhcpd.conf" << 'EOF'
# Configuration file for ISC 2.0 vmnet-dhcpd operating on vmnet8.

###### VMNET DHCP Configuration. Start of "DO NOT MODIFY SECTION" #####
# Written by vmware-fusion-nat-to-192.168.56.sh
allow unknown-clients;
default-lease-time 1800;
max-lease-time 7200;

subnet 192.168.56.0 netmask 255.255.255.0 {
	range 192.168.56.128 192.168.56.254;
	option broadcast-address 192.168.56.255;
	option domain-name-servers 192.168.56.2;
	option domain-name localdomain;
	default-lease-time 1800;
	max-lease-time 7200;
	option netbios-name-servers 192.168.56.2;
	option routers 192.168.56.2;
}
host vmnet8 {
	hardware ethernet 00:50:56:C0:00:08;
	fixed-address 192.168.56.1;
	option domain-name-servers 0.0.0.0;
	option domain-name "";
	option routers 0.0.0.0;
}
####### VMNET DHCP Configuration. End of "DO NOT MODIFY SECTION" #######
EOF

echo "完成。备份: networking.bak.$TS 等。请重启 VMware Fusion（或重启 Mac），再在 Ubuntu 中确认："
echo "  地址 192.168.56.11/24，网关 192.168.56.2，DNS 可 8.8.8.8"
echo "Mac 上应出现 bridge 接口 inet 192.168.56.1，再 ping 192.168.56.11"
