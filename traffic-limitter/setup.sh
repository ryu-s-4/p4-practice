#!/bin/bash

# veth for each hosts (must be odd number)
VETH="1 3 5 7"

if [ ${EUID:-${UID}} != 0 ]; then 
	echo "ERROR : should execute as the root user."
	exit 1
fi

function create () {

	for i in $VETH
	do
		host="host"$i
		veth="veth"$i
		addr="192.168.0."$i"/24"

		if [ -z "`ip netns show | grep $host`" ]; then
			
			# create host
			ip netns add $host
			echo "INFO: host$i (netns) is creaed." 
		
			# attach veth to the host.
			ip link set $veth netns $host up

			# link up
			ip netns exec $host ip link set dev $veth up
			ip netns exec $host ip link set dev lo up

			# allocate IP addr.
			ip netns exec $host ip a add $addr dev $veth

			echo "INFO: $host is setup with $veth."
		else
			echo "INFO: host$i is already exist."
		fi
	done
}

function destroy () {

	# remove the interface / delete the host
	for i in $VETH
	do
		host="host"$i
		veth="veth"$i
		if [ -n "`ip netns show | grep $host`" ]; then
			ip netns exec $host ip link set $veth netns 1
			ip link set dev $veth up
			ip netns delete $host
			echo "INFO: $host is deleted."
		else
			echo "INFO $host does not exist."
		fi
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