/* -*- P4_16 -*- */
#include <core.p4>
#include <v1model.p4>

const bit<16> TYPE_IPV4 = 0x0800;
const bit<16> TYPE_ARP  = 0x0806;
const bit<16> TYPE_VLAN = 0x8100;

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

struct metadata {
    bit<2> color;
    bit<32> cnt_idx;
}

struct headers {
    ethernet_t  ethernet;
    arp_t        arp;
    vlan_t       vlan;
    ipv4_t       ipv4;
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

    const bit<32> CNT_SIZE = 1024;
    counter(CNT_SIZE, CounterType.bytes) traffic_cnt;

    /* mac_exact table と紐付けることで
       宛先 MAC Addr.（ table の key）毎に
       入力レートを計測 = レート制御が可能 */
    direct_meter<bit<2>>(MeterType.packets) dm;

    action drop() {
        // mark_to_drop is NOT implemented on BMv2.
        // mark_to_drop(standard_metadata);
    }
    
    action broadcast() {
        /* multicast group を c-plane で定義し、
           対応する group の id を mcast_grp に入力.
           詳細は下記 URL の "after-ingress-pseud code" を参照. 
           参考: https://github.com/p4lang/behavioral-model/blob/master/docs/simple_switch.md */

        /* 現状の BMv2 では CLI 経由での multicast group の登録に対応していない可能性あり.
           tutorial は runtime.json に直書きしているので、最悪そうする. */

        /* mcast_grp == 1 は全てのポートが属する broadcast domain として default 登録（とする） */
        standard_metadata.mcast_grp = 1;
    }

    action broadcast_vlan(bit<16> grp_id) {
        /* grp_id <- vlan_id として broadcast domain を C/P より登録.
           ただし、stateful な実装は気持ち悪いため mcast_grp は別途 action-param として登録 */
        standard_metadata.mcast_grp = grp_id;

        // meta.cnt_idx = (bit<32>) hdr.vlan.id;
        traffic_cnt.count(meta.cnt_idx);
    }

    action switching(egressSpec_t port) {
        /* 入力レートの判定値を取得 */
        dm.read(meta.color);

        /* color が RED or YELLOW の場合はドロップ
        　 　RED     : peak rate を超過した場合
        　　 YELLOW :  committed rate を超過した場合 */
        if (meta.color == V1MODEL_METER_COLOR_YELLOW) {
            if (meta.color == V1MODEL_METER_COLOR_RED) {
                drop();
            } else {
                drop();
            }
        } else {
        /* 出力ポートを設定 */
        standard_metadata.egress_spec = port;
        }
    }

    action switching_vlan(egressSpec_t port) {
        /* 出力ポートを設定 */
        standard_metadata.egress_spec = port;

        /* 転送したトラヒックをカウント */
        meta.cnt_idx = (bit<32>) hdr.vlan.id;
        traffic_cnt.count(meta.cnt_idx);
    }
  
    table mac_exact {
        key = {
            hdr.ethernet.dstAddr: exact;
        }
        actions = {
            switching;
            broadcast;
            drop;
            NoAction();
        }
        size = 1024;
        default_action = drop;
        meters = dm;
    }

    table mac_vlan_exact {
        key = {
            hdr.vlan.id: exact;
            hdr.ethernet.dstAddr: exact;
        }
        actions = {
            switching_vlan;
            broadcast_vlan;
            drop;
            NoAction();
        }
        size = 1024;
        default_action = drop;
    }
    
    apply {
        if (hdr.vlan.isValid()) {
            meta.cnt_idx = (bit<32>) hdr.vlan.id;
            if (!mac_vlan_exact.apply().hit) {
                /* MAC 学習 */
            }
        } else {
            if (!mac_exact.apply().hit) {
                /* MAC 学習 */
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
            /* mcast_grp = 1 には全てのポートが参加しているため、入力ポートへの送信は防ぐ */
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
