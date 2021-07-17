/*
SRv6 (RFC8986) implementation.
Supported functions are as follows.
 - End, End.X, End.DX2, End.DX2V, End.DT2U, End.DT2M
 - H.Encaps

This file MUST be included to the main program.
*/

/********* HEADER ***********/
#define LABELS 3      // Number of SRv6 SID stacks
#define CPU_PORT 9w192  // port num. for CPU forwarding

const bit<16> ETYPE_IPv4 = 0x0800;
const bit<16> ETYPE_IPv6 = 0x86dd;
const bit<16> ETYPE_VLAN = 0x8100;

const bit<8> PROTOCOL_IPIP   = 8w4;
const bit<8> PROTOCOL_IPv6   = 8w41;
const bit<8> PROTOCOL_SRH    = 8w43;
const bit<8> PROTOCOL_ETHER  = 8w143;

const bit<16> INIT      = 16w0;
const bit<16> END       = 16w2;
const bit<16> END_X     = 16w3;
const bit<16> END_DX2   = 16w5;
const bit<16> END_DX2V  = 16w6;
const bit<16> END_DT2U  = 16w7;
const bit<16> END_DT2M  = 16w8;

typedef bit<48>  mac_addr_t;
typedef bit<12>  vlan_id_t;
typedef bit<32>  ipv4_addr_t;
typedef bit<128> ipv6_addr_t;
typedef bit<16>  port_p4rt_t;

header ethernet_h {
    mac_addr_t dst_addr;
    mac_addr_t src_addr;
    bit<16> ether_type;
}

header vlan_tag_h { 
    bit<3> pcp;
    bit<1> cfi;
    vlan_id_t id;
    bit<16> ether_type;
}

header ipv4_h {
    bit<4> version;
    bit<4> ihl;
    bit<8> diffserv;
    bit<16> total_len;
    bit<16> identification;
    bit<3> flags;
    bit<13> flag_offset;
    bit<8> ttl;
    bit<8> protocol;
    bit<16> hdr_checksum;
    ipv4_addr_t src_addr;
    ipv4_addr_t dst_addr;
}

header ipv6_h {
    bit<4> version;
    bit<8> traffic_class;
    bit<20> flow_label;
    bit<16> payload_len;
    bit<8> next_hdr;
    bit<8> hop_limit;
    ipv6_addr_t src_addr;
    ipv6_addr_t dst_addr;
}

header ipv6_srh_h {
    bit<8> next_hdr;
    bit<8> hdr_ext_len;
    bit<8> routing_type;
    bit<8> seg_left;
    bit<8> last_entry;
    bit<8> flags;
    bit<16> tag;
}

header sid_h {
    bit<128> sid;
}

@controller_header("packet_in")
header packet_in_h {
    port_p4rt_t ingress_port;
}

struct header_t {
    packet_in_h   packetIn;
    ethernet_h    ethernet;
    vlan_tag_h    vlan; 
    ipv4_h        ipv4;
    ipv6_h        ipv6;
    ipv6_srh_h    srh;
    sid_h[LABELS] sid_stack;
    ethernet_h    inner_ether;
    vlan_tag_h    inner_vlan;
    ipv4_h        inner_ipv4;
}

struct metadata_t {
    bool    psp_flag;
    bool    upper_layer_proc_flag;
    bool    ipv4_checksum_needed;
    bool    l2_forward_flag;
    bool    vlan_forward_flag;
    bool    ipv4_forward_flag;
    bool    ipv6_forward_flag;
    bit<64> rd;
    bit<12> vlan_id;
    bit<16> payload_len;
    bit<16> srv6_func_applied;
    mac_addr_t  ethernet_dst_addr;
    ipv4_addr_t ipv4_dst_addr;
    ipv6_addr_t ipv6_dst_addr;
    ipv6_addr_t next_sid;
}