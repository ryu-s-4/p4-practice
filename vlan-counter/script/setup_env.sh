#!/bin/bash

# create host(netns)
ip netns add host1
ip netns add host3
ip netns add host5
ip netns add host7

# attach interface to host
ip link set veth1 netns host1 up
ip link set veth3 netns host3 up
ip link set veth5 netns host5 up
ip link set veth7 netns host7 up

# link-up
ip netns exec host1 ip link set dev veth1 up
ip netns exec host3 ip link set dev veth3 up
ip netns exec host5 ip link set dev veth5 up
ip netns exec host7 ip link set dev veth7 up

# allocate IP address
ip netns exec host1 ip a add 192.168.0.1/24 dev veth1
ip netns exec host3 ip a add 192.168.0.3/24 dev veth3
ip netns exec host5 ip a add 192.168.0.5/24 dev veth5
ip netns exec host7 ip a add 192.168.0.7/24 dev veth7

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
