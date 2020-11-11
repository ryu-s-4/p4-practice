/* -*- P4_16 -*- */
#include <core.p4>
#include <v1model.p4>

const bit<16> TYPE_IPV4 = 0x0800;
const bit<16> TYPE_ARP  = 0x0806;
const bit<16> TYPE_VLAN = 0x8100;

const bit<8> IPPROTO_UDP = 17;

const bit<16> PORT_GTPU = 2152;

/*************************************************************************
*********************** H E A D E R S  ***********************************
*************************************************************************/

typedef bit<9>  egressSpec_t;
typedef bit<48> macAddr_t;
typedef bit<32> ip4Addr_t;

header ethernet_t {
    macAddr_t dstAddr;
    macAddr_t srcAddr;
    bit<16>   etherType;
}

header vlan_t {
    bit<3> priority;
    bit<1> dei;
    bit<12> id;
    bit<16> etherType;
}

header arp_t {
    bit<16> hardtType;
    bit<16> protoType;
    bit<8> hardSize;
    bit<8> protoSize;
    bit<16> opcode;
    macAddr_t send_macAddr;
    ip4Addr_t send_ip4Addr;
    macAddr_t trgt_macAddr;
    ip4Addr_t trgt_ip4Addr;
}

header ipv4_t {
    bit<4>    version;
    bit<4>    ihl;
    bit<8>    diffserv;
    bit<16>   totalLen;
    bit<16>   identification;
    bit<3>    flags;
    bit<13>   fragOffset;
    bit<8>    ttl;
    bit<8>    protocol;
    bit<16>   hdrChecksum;
    ip4Addr_t srcAddr;
    ip4Addr_t dstAddr;
}

header udp_t {
    bit<16> srcPort;
    bit<16> dstPort;
    bit<16> len;
    bit<16> checksum;
}

header gtp_u_t {
    bit<3> ver;
    bit<1> ptype;
    bit<1> resv;
    bit<1> ehflag;
    bit<1> snflag;
    bit<1> nnflag;
    bit<8> mtype;
    bit<16> mlen;
    bit<32> teid;
    bit<16> seq;
    bit<8> npdu;
    bit<8> ext_htype;
}

struct metadata {
    bit<2> color;
    bool drop_flag;
}

struct headers {
    ethernet_t  ethernet;
    arp_t       arp;
    vlan_t      vlan;
    ipv4_t      ipv4;
    udp_t       udp;
    gtp_u_t     gtp_u;
}

/*************************************************************************
*********************** P A R S E R  ***********************************
*************************************************************************/

parser MyParser(packet_in packet,
                out headers hdr,
                inout metadata meta,
                inout standard_metadata_t standard_metadata) {

    state start {
        transition parse_ethernet;
    }

    state parse_ethernet {
        packet.extract(hdr.ethernet);
        transition select(hdr.ethernet.etherType) {
            TYPE_VLAN : parse_vlan;
            TYPE_ARP  : parse_arp;
            TYPE_IPV4 : parse_ipv4;
            default   : accept;
        }
    }

    state parse_vlan {
        packet.extract(hdr.vlan);
        transition select(hdr.vlan.etherType) {
            TYPE_ARP  : parse_arp;
            TYPE_IPV4 : parse_ipv4;
            default   : accept; 
        }
    }

    state parse_arp {
        packet.extract(hdr.arp);
        transition accept;
    }

    state parse_ipv4 {
        packet.extract(hdr.ipv4);
        transition select(hdr.ipv4.protocol) {
            IPPROTO_UDP : parse_udp;
            default : accept;
        }
    }

    state parse_udp {
        packet.extract(hdr.udp);
        transition select(hdr.udp.dstPort) {
            PORT_GTPU : parse_gtp_u;
            default : accept;
        }
    }

    state parse_gtp_u {
        packet.extract(hdr.gtp_u);
        transition accept;
    }
}

/*************************************************************************
************   C H E C K S U M    V E R I F I C A T I O N   *************
*************************************************************************/

control MyVerifyChecksum(inout headers hdr, inout metadata meta) {   
    apply {  }
}


/*************************************************************************
**************  I N G R E S S   P R O C E S S I N G   *******************
*************************************************************************/

control MyIngress(inout headers hdr,
                  inout metadata meta,
                  inout standard_metadata_t standard_metadata) {

    /*
    const bit<32> CNT_SIZE = 4096;
    counter(CNT_SIZE, CounterType.packets) traffic_cnt;
    */

    const bit<32> CNT_SIZE = 0xffffffff;
    counter(CNT_SIZE, CounterType.bytes) meter_cnt;
    direct_meter<bit<2>>(MeterType.bytes) limitter;
    
    action drop() {
        mark_to_drop(standard_metadata);
    }

    action broadcast() {
        standard_metadata.mcast_grp = 1;
    }

    action broadcast_vlan(bit<16> grp_id) {
        standard_metadata.mcast_grp = grp_id;
    }

    action switching(egressSpec_t port) {
        standard_metadata.egress_spec = port;
    }

    action limit_traffic() {
        /* hit したエントリの (direct) meter を確認し超過している場合は速度制限 */
        limitter.read(meta.color);
        if (meta.color == V1MODEL_METER_COLOR_RED) {
            meta.drop_flag = true;
        }
    }
  
    table mac_exact {
        key = {
            hdr.ethernet.dstAddr: exact;
        }
        actions = {
            switching;
            broadcast;
            drop;
        }
        size = 1024;
        default_action = drop;
    }

    table mac_vlan_exact {
        key = {
            hdr.vlan.id: exact;
            hdr.ethernet.dstAddr: exact;
        }
        actions = {
            switching;
            broadcast_vlan;
            drop;
        }
        size = 1024;
        default_action = drop;
    }

    table urr_exact {
        key = { hdr.gtp_u.teid : exact; }
        actions = { 
            limit_traffic;
            NoAction();
        }
        meters = limitter;
        default_action = NoAction();
    }
    
    apply {

        if (hdr.gtp_u.isValid()) {
            /* TEID 毎にトラヒック量監視 */
            meter_cnt.count(hdr.gtp_u.teid);

            /* 速度制限有無を確認 */
            meta.drop_flag = false;
            urr_exact.apply();
            if (meta.drop_flag == true) {
                drop();
            }
        }

        if (hdr.vlan.isValid()) {
            if (!mac_vlan_exact.apply().hit) {
                /* TODO: MAC learning */
            }
        } else {
            if (!mac_exact.apply().hit) {
                /* TODO: MAC learning */
            }
        }
    }
}

/*************************************************************************
****************  E G R E S S   P R O C E S S I N G   *******************
*************************************************************************/

control MyEgress(inout headers hdr,
                 inout metadata meta,
                 inout standard_metadata_t standard_metadata) {

    action drop() {
        mark_to_drop(standard_metadata);
    }

    apply {  
        if (standard_metadata.egress_port == standard_metadata.ingress_port) {
            drop();
        }
    }
}

/*************************************************************************
*************   C H E C K S U M    C O M P U T A T I O N   **************
*************************************************************************/

control MyComputeChecksum(inout headers hdr, inout metadata meta) {
     apply {
	update_checksum(
	    hdr.ipv4.isValid(),
            { hdr.ipv4.version,
	      hdr.ipv4.ihl,
              hdr.ipv4.diffserv,
              hdr.ipv4.totalLen,
              hdr.ipv4.identification,
              hdr.ipv4.flags,
              hdr.ipv4.fragOffset,
              hdr.ipv4.ttl,
              hdr.ipv4.protocol,
              hdr.ipv4.srcAddr,
              hdr.ipv4.dstAddr },
            hdr.ipv4.hdrChecksum,
            HashAlgorithm.csum16);
    }
}

/*************************************************************************
***********************  D E P A R S E R  *******************************
*************************************************************************/

control MyDeparser(packet_out packet, in headers hdr) {
    apply {
        packet.emit(hdr.ethernet);
        packet.emit(hdr.vlan);
        packet.emit(hdr.arp);
        packet.emit(hdr.ipv4);
    }
}

/*************************************************************************
***********************  S W I T C H  *******************************
*************************************************************************/

V1Switch(
MyParser(),
MyVerifyChecksum(),
MyIngress(),
MyEgress(),
MyComputeChecksum(),
MyDeparser()
) main;
