#!/bin/bash

# veth for each hosts (must be odd number)
VETH="1 13 15 17"

if [ ${EUID:-${UID}} != 0 ]; then 
	echo "ERROR : should execute as the root user."
	exit 1
fi

function create () {

	# load kernel module if not yet.
	if [[ `lsmod` != *8021q* ]]; then
		modprobe 8021q
	fi
	
	idx=1
	for i in $VETH
	do
		host="host"$idx
		veth="veth"$i
		veth_10=$veth".10"
		veth_20=$veth".20"
		addr_10="192.168.10."$i"/24"
		addr_20="192.168.20."$i"/24"

		if [ -z "`ip netns show | grep $host`" ]; then
			
			# create host
			ip netns add $host
			echo "INFO: host$idx (netns) is creaed." 
		
			# attach veth to the host.
			ip link set $veth netns $host up

			# set up VLAN interface for each host
			if [ $idx == 1 ]; then

				ip netns exec $host ip link add link $veth name $veth_10 type vlan id 10
				ip netns exec $host ip link add link $veth name $veth_20 type vlan id 20

				ip netns exec $host ip a add $addr_10 dev $veth_10
				ip netns exec $host ip a add $addr_20 dev $veth_20

				ip netns exec $host ip link set dev $veth_10 up
				ip netns exec $host ip link set dev $veth_20 up

			elif [ $idx == 4 ]; then

				ip netns exec $host ip link add link $veth name $veth_20 type vlan id 20
				ip netns exec $host ip a add $addr_20 dev $veth_20
				ip netns exec $host ip link set dev $veth_20 up

			else

				ip netns exec $host ip link add link $veth name $veth_10 type vlan id 10
				ip netns exec $host ip a add $addr_10 dev $veth_10
				ip netns exec $host ip link set dev $veth_10 up

			fi
			echo "INFO: $host is setup with $veth."
			
		else
			echo "INFO: host$i is already exist."
		fi

		idx=`expr $idx + 1`
	done
}

function destroy () {

	# remove the interface / delete the host
	idx=1
	for i in $VETH
	do
		host="host"$idx
		veth="veth"$i
		if [ -n "`ip netns show | grep $host`" ]; then
			ip netns exec $host ip link set $veth netns 1
			ip link set dev $veth up
			ip netns delete $host
			echo "INFO: $host is deleted."
		else
			echo "INFO $host does not exist."
		fi
		idx=`expr $idx + 1`
	done
}

while getopts "cd" OPT;
do
	case $OPT in
		c ) create
			exit 1;;
		d ) destroy
			exit 1;;
	esac
done

echo "Usage: $0 [-c|-d] (create or destroy)"