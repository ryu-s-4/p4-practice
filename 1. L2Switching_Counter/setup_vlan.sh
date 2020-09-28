#!/bin/bash

# load kernel module if not yet.
if [[ `lsmod` != *8021q* ]]; then
	modprobe 8021q
fi

# create vlan iterface 
ip netns exec host1 ip link add link veth1 name veth1.100 type vlan id 100
ip netns exec host1 ip link add link veth1 name veth1.200 type vlan id 200
ip netns exec host3 ip link add link veth3 name veth3.200 type vlan id 200
ip netns exec host5 ip link add link veth5 name veth5.100 type vlan id 100
ip netns exec host7 ip link add link veth7 name veth7.200 type vlan id 200

# link-up 
ip netns exec host1 ip link set dev veth1.100 link up
ip netns exec host1 ip link set dev veth1.200 link up
ip netns exec host3 ip link set dev veth3.200 link up
ip netns exec host5 ip link set dev veth5.100 link up
ip netns exec host7 ip link set dev veth7.200 link up

# allocate IP address
ip netns exec host1 ip a add 192.168.100.1/24 dev veth1.100
ip netns exec host1 ip a add 192.168.200.1/24 dev veth1.200
ip netns exec host3 ip a add 192.168.200.3/24 dev veth3.200
ip netns exec host5 ip a add 192.168.100.5/24 dev veth5.100
ip netns exec host7 ip a add 192.168.200.7/24 dev veth7.200
