#!/bin/bash

# change the directory to the one that gtp-tunnel exist.
DIR="/home/p4/p4-practice/traffic-limitter/libgtpnl/tools"

# Host IDs for each tunnel "uplink downlink" 
# HOST_TUN1="1 3" # indicates that host1 -> host3 is uplink and vise varse for tunnel1
# HOST_TUN2="5 7" # indicates that host5 -> host7 is uplink and vise varse for tunnel2

# veth for each hosts of a tunnel (must be odd number)
VETH="1 3"
# VETH_TUN2="5 7"

# TEIDs for each tunnel
TEID="100 200"
# TEID_TUN2="200 250"

if [ ${EUID:-${UID}} != 0 ]; then 
	echo "ERROR : should execute as the root user."
	exit 1
fi

function create_hosts () {

	# create hosts (netns).
	for i in $VETH
	do
		host="host"$i
		if [ -z "`ip netns show | grep $host`" ]; then
			echo "INFO: host$i (netns) is creaed." 
			ip netns add $host
		else
			echo "INFO: host$i is already exist."
		fi
	done
}

function check_gtp_if () {

	# check the existance of gtp interfaces.
	for i in $VETH
	do
		host="host"$i
		gtpif="gtp"$i
		if [[ -z `ip netns exec $host ip link show dev $gtpif` ]]; then
			echo "ERROR: Not created $gtpif with \"gtp-link add $gtpif\""
			exit 1
		fi
	done
}

function create_network () {

	# create network.
	for i in $VETH
	do
		host="host"$i
		veth="veth"$i
		addr="192.168.0."$i"/24"
		addr_lo="192.168.99."$i"/32"

		# attach veth to the host.
		ip link set $veth netns $host up

		# link up
		ip netns exec $host ip link set dev $veth up
		ip netns exec $host ip link set dev lo up

		# allocate IP addr.
		ip netns exec $host ip a add $addr dev $veth

		# allocate IP addr. for lo.
		ip netns exec $host ip a add $addr_lo dev lo

		# disable checksum offload
		sudo ip netns exec $host ethtool --offload $veth rx off tx off
	done
}

function establish_tunnel () {

	cnt=0
	for i in $VETH
	do
		if [ $cnt -eq 0 ]; then
			veth_down=$i
		else
			veth_up=$i
		fi
		cnt=$(($cnt+1))
	done

	cnt=0
	for i in $TEID
	do
		if [ $cnt -eq 0 ]; then
			teid_uplink=$i
		else
			teid_downlink=$i
		fi
		cnt=$(($cnt+1))
	done

	# configuration at down-side
	host="host"$veth_down
	gtpif="gtp"$veth_down
	addr="192.168.0."$veth_up
	addr_lo="192.168.99."$veth_up

	ip netns exec $host $DIR/gtp-tunnel add $gtpif v1 $teid_downlink $teid_uplink $addr_lo $addr
	ip netns exec $host ip route add $addr_lo/32 dev $gtpif

	# configuration at up-side
	host="host"$veth_up
	gtpif="gtp"$veth_up
	addr="192.168.0."$veth_down
	addr_lo="192.168.99."$veth_down

	ip netns exec $host $DIR/gtp-tunnel add $gtpif v1 $teid_uplink $teid_downlink $addr_lo $addr
	ip netns exec $host ip route add $addr_lo/32 dev $gtpif
}

function destroy () {

	# delete the hosts.
	for i in $VETH
	do
		host="host"$i
		ip netns delete $host
	done
}

#destroy

#create_hosts
#create_network

#check_gtp_if

establish_tunnel

exit 0










function old_prepare_params () {

	# prepare the array of host ids
	for i in `seq 
	if [ -v TUN_PARAM$i ]; then

	for id in $HOST_TUN1
	do
		HOST[$CNT]=$id
		CNT=$CNT+1
	done
	for id in $HOST_TUN2
	do
		HOST[$CNT]=$id
		CNT=$CNT+1
	done

	# prepare the array of TEIDs
	CNT=0
	for id in $TEID_TUN1
	do
		TEID[$CNT]="$id"
		CNT=$CNT+1
	done
	for id in $TEID_TUN2
	do
		TEID[$CNT]="$id"
		CNT=$CNT+1
	done

	# prepare veth device name
	CNT=0
	for id in 1 3 5 7
	do
		VETH[$CNT]="veth"$id
		CNT=$CNT+1
	done
}

function old_create_hosts () {

	# create hosts (netns) if not.
	for i in `seq 0 3`
	do
		id=${HOST[$i]}
		host="host"$id
		if [ -z "`ip netns show | grep $host`" ]; then
			echo "INFO: host$id (netns) is creaed." 
			ip netns add $host
		else
			echo "INFO: host$id is already exist."
		fi
	done
}