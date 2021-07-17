/*
SRv6 (RFC8986) implementation.
Supported functions are as follows.
 - End, End.X, End.DX2, End.DX2V, End.DT2U, End.DT2M
 - H.Encaps
*/

#include<core.p4>
#include<v1model.p4>
#include "headers.p4"

/*********** PARSER **********/
parser SwitchParser(packet_in pkt,
            		out header_t hdr,
                	inout metadata_t meta,
                	inout standard_metadata_t std_meta)
{

    state start {

        // initialize metadata
        meta.psp_flag              = false;
        meta.upper_layer_proc_flag = false;
        meta.ipv4_checksum_needed  = false;
        meta.l2_forward_flag       = false;
        meta.vlan_forward_flag     = false;
        meta.ipv4_forward_flag     = false;
        meta.ipv6_forward_flag     = false;
        meta.rd                    = 64w0;
        meta.srv6_func_applied     = INIT;
        meta.payload_len           = (bit<16>)std_meta.packet_length;

        transition parse_ethernet;
    }

    state parse_ethernet {
        pkt.extract(hdr.ethernet);
        meta.payload_len = meta.payload_len - 14;
        transition select(hdr.ethernet.ether_type) {
			ETYPE_VLAN : parse_vlan;
            ETYPE_IPv4 : parse_ipv4;
            ETYPE_IPv6 : parse_ipv6;
            default    : accept;
        }
    }

    state parse_vlan {
		pkt.extract(hdr.vlan);
        meta.payload_len = meta.payload_len - 4;
		transition select(hdr.vlan.ether_type) {
			ETYPE_IPv4 : parse_ipv4;
            ETYPE_IPv6 : parse_ipv6;
            default    : accept;
		}
	}

    state parse_ipv4 {
        pkt.extract(hdr.ipv4);
        meta.payload_len = meta.payload_len - 20;
        transition accept;
    }

    state parse_ipv6 {
        pkt.extract(hdr.ipv6);
        meta.payload_len = meta.payload_len - 40;
        transition select(hdr.ipv6.next_hdr) {
            PROTOCOL_SRH   : parse_srh;
			PROTOCOL_ETHER : parse_inner_ether;
            default        : accept;        
        }
    }

    state parse_srh {
        pkt.extract(hdr.srh);
        meta.payload_len = meta.payload_len - 8;
        transition select(hdr.srh.last_entry) {
            0 : parse_sid_stack_1;
            1 : parse_sid_stack_2;
            2 : parse_sid_stack_3;
            // 3 : parse_sid_stack_4;
            default : check_srh_next_hdr;
        }
    }

    state parse_sid_stack_1 {
        pkt.extract(hdr.sid_stack.next);
        meta.payload_len = meta.payload_len - 16;
        transition check_srh_next_hdr;
    }

    state parse_sid_stack_2 {
        pkt.extract(hdr.sid_stack.next);
        pkt.extract(hdr.sid_stack.next);
        meta.payload_len = meta.payload_len - 32;
        transition check_srh_next_hdr;
    }

    state parse_sid_stack_3 {
        pkt.extract(hdr.sid_stack.next);
        pkt.extract(hdr.sid_stack.next);
        pkt.extract(hdr.sid_stack.next);
        meta.payload_len = meta.payload_len - 48;
        transition check_srh_next_hdr;
    }

    /*
    state parse_sid_stack_4 {
        pkt.extract(hdr.sid_stack.next);
        pkt.extract(hdr.sid_stack.next);
        pkt.extract(hdr.sid_stack.next);
        pkt.extract(hdr.sid_stack.next);
        transition check_srh_next_hdr;
    }
    */

    state check_srh_next_hdr {
        transition select(hdr.srh.next_hdr) {
            PROTOCOL_ETHER : parse_inner_ether;
            default        : accept;
        }
    }

    state parse_inner_ether {
        pkt.extract(hdr.inner_ether);
        meta.payload_len = meta.payload_len - 14;
        transition select(hdr.inner_ether.ether_type) {
            ETYPE_IPv4 : parse_inner_ipv4;
            ETYPE_VLAN : parse_inner_vlan;
            default    : accept;
        }
    }

    state parse_inner_vlan {
        pkt.extract(hdr.inner_vlan);
        meta.payload_len = meta.payload_len - 4;
        transition select(hdr.inner_vlan.ether_type) {
            ETYPE_IPv4 : parse_inner_ipv4;
            default    : accept;
        }
    }

    state parse_inner_ipv4 {
        pkt.extract(hdr.inner_ipv4);
        meta.payload_len = meta.payload_len - 20;
        transition accept;
    }
}

/*********** VERIFY CHECKSUM **********/
control SwitchVerifyChecksum(inout header_t hdr, inout metadata_t meta)
{   
    apply {}
}

/********** INGRESS **********/
control SwitchIngress(inout header_t hdr,
                	  inout metadata_t meta,
                	  inout standard_metadata_t std_meta)
{

    action h_encaps_0_inner_ether(bit<128> src_addr, bit<128> sid_0) {

        // copy outer ethernet header to inner ether header
        // this is equal to encapsulate outer ethernet header
        hdr.inner_ether.setValid();
        hdr.inner_ether.dst_addr = hdr.ethernet.dst_addr;
        hdr.inner_ether.src_addr = hdr.ethernet.src_addr;
        hdr.inner_ether.ether_type = hdr.ethernet.ether_type;
        hdr.ethernet.ether_type = ETYPE_IPv6;

        // encapsulate with IPv6, SRH, SID stacks
        hdr.ipv6.setValid();
        hdr.srh.setValid();
        hdr.sid_stack[0].setValid();

        hdr.ipv6.version = 6;
        hdr.ipv6.traffic_class = 0;
        hdr.ipv6.flow_label = 0;
        hdr.ipv6.payload_len = meta.payload_len + 38; // inner_ether, length of SRH, SID * 1, payload
        hdr.ipv6.next_hdr = PROTOCOL_SRH;
        hdr.ipv6.hop_limit = 255;
        hdr.ipv6.src_addr = src_addr;
        hdr.ipv6.dst_addr = sid_0;

        hdr.srh.next_hdr = PROTOCOL_ETHER;
        hdr.srh.hdr_ext_len = 16;
        hdr.srh.routing_type = 4;
        hdr.srh.seg_left = 0;
        hdr.srh.last_entry = 0;
        hdr.srh.flags = 0;
        hdr.srh.tag = 0;

        hdr.sid_stack[0].sid = sid_0;
    }

    action h_encaps_1_inner_ether(bit<128> src_addr, bit<128> sid_0, bit<128> sid_1) {

        // copy outer ethernet header to inner ether header
        // this is equal to encapsulate outer ethernet header
        hdr.inner_ether.setValid();
        hdr.inner_ether.dst_addr = hdr.ethernet.dst_addr;
        hdr.inner_ether.src_addr = hdr.ethernet.src_addr;
        hdr.inner_ether.ether_type = hdr.ethernet.ether_type;
        hdr.ethernet.ether_type = ETYPE_IPv6;
        
        // encapsulate with IPv6, SRH, SID stacks
        hdr.ipv6.setValid();
        hdr.srh.setValid();
        hdr.sid_stack[0].setValid();
        hdr.sid_stack[1].setValid();

        hdr.ipv6.version = 6;
        hdr.ipv6.traffic_class = 0;
        hdr.ipv6.flow_label = 0;
        hdr.ipv6.payload_len = meta.payload_len + 54; // length of inner_ether, SRH, SID * 2, payload
        hdr.ipv6.next_hdr = PROTOCOL_SRH;
        hdr.ipv6.hop_limit = 255;
        hdr.ipv6.src_addr = src_addr;
        hdr.ipv6.dst_addr = sid_1;

        hdr.srh.next_hdr = PROTOCOL_ETHER;
        hdr.srh.hdr_ext_len = 16;
        hdr.srh.routing_type = 4;
        hdr.srh.seg_left = 1;
        hdr.srh.last_entry = 1;
        hdr.srh.flags = 0;
        hdr.srh.tag = 0;

        hdr.sid_stack[0].sid = sid_0;
        hdr.sid_stack[1].sid = sid_1;
    }

    action h_encaps_2_inner_ether(bit<128> src_addr, bit<128> sid_0, bit<128> sid_1, bit<128> sid_2) {

        // copy outer ethernet header to inner ether header
        // this is equal to encapsulate outer ethernet header
        hdr.inner_ether.setValid();
        hdr.inner_ether.dst_addr = hdr.ethernet.dst_addr;
        hdr.inner_ether.src_addr = hdr.ethernet.src_addr;
        hdr.inner_ether.ether_type = hdr.ethernet.ether_type;
        hdr.ethernet.ether_type = ETYPE_IPv6;
       
        // encapsulate with IPv6, SRH, SID stacks
        hdr.ipv6.setValid();
        hdr.srh.setValid();
        hdr.sid_stack[0].setValid();
        hdr.sid_stack[1].setValid();
        hdr.sid_stack[2].setValid();

        hdr.ipv6.version = 6;
        hdr.ipv6.traffic_class = 0;
        hdr.ipv6.flow_label = 0;
        hdr.ipv6.payload_len = meta.payload_len + 70; // length of inner_ether, SRH, SID * 3, payload
        hdr.ipv6.next_hdr = PROTOCOL_SRH;
        hdr.ipv6.hop_limit = 255;
        hdr.ipv6.src_addr = src_addr;
        hdr.ipv6.dst_addr = sid_2;

        hdr.srh.next_hdr = PROTOCOL_ETHER;
        hdr.srh.hdr_ext_len = 16;
        hdr.srh.routing_type = 4;
        hdr.srh.seg_left = 2;
        hdr.srh.last_entry = 2;
        hdr.srh.flags = 0;
        hdr.srh.tag = 0;

        hdr.sid_stack[0].sid = sid_0;
        hdr.sid_stack[1].sid = sid_1;
        hdr.sid_stack[2].sid = sid_2;
    }

    table srv6_encaps_func {
        key = {
            hdr.ethernet.dst_addr : ternary;
            hdr.ethernet.src_addr : ternary;
            hdr.vlan.id           : ternary;
            hdr.ipv4.dst_addr     : ternary;
            hdr.ipv4.src_addr     : ternary;
        }
        actions = {
            h_encaps_0_inner_ether;
            h_encaps_1_inner_ether;
            h_encaps_2_inner_ether;
            NoAction;
        }
        default_action = NoAction;
        size = 1024;
    }

    action forwarding_to_CPU() {

        // set forwarding port to CPU
        std_meta.egress_spec = CPU_PORT;
        hdr.packetIn.setValid();
        hdr.packetIn.ingress_port = ((port_p4rt_t)(std_meta.ingress_port));

        /* remove all headers (expose application packet) */
        hdr.ethernet.setInvalid();
        hdr.vlan.setInvalid();
        hdr.ipv4.setInvalid();
        hdr.ipv6.setInvalid();
        hdr.srh.setInvalid();
        hdr.sid_stack[0].setInvalid();
        hdr.sid_stack[1].setInvalid();
        hdr.sid_stack[2].setInvalid();
        hdr.inner_ether.setInvalid();
        hdr.inner_vlan.setInvalid();
        hdr.inner_ipv4.setInvalid();
    }

    table l2_check {
        key = {
            std_meta.ingress_port : ternary;
            hdr.vlan.id           : ternary;
            hdr.ethernet.dst_addr : exact;
        }
        actions = {
            forwarding_to_CPU;
            NoAction;
        }
        default_action = NoAction;
        size = 1024;
    }

    action multicast_default() {

        // set multicast group id with least significant bits of RD
        std_meta.mcast_grp = meta.rd[15:0]+1;
    }

    action switching(bit<9> port) {

        // set forwarding port
        std_meta.egress_spec = port;
    }

    table l2_forward {
        key = {
            meta.rd                : exact;
            meta.ethernet_dst_addr : exact;
        }
        actions = {
            multicast_default;
            switching;
        }
        default_action = multicast_default();
        size = 1024;
    }

    table vlan_forward {
        key = {
            meta.rd      : exact;
            meta.vlan_id : exact;
        }
        actions = {
            switching;
            NoAction;
        }
        default_action = NoAction;
        size = 1024;
    }

    action forwarding_v4(bit<9> port, mac_addr_t dst, mac_addr_t src) {
        
        // set forwarding port
        std_meta.egress_spec = port;

        // update MAC addr
        hdr.ethernet.src_addr = src;
        hdr.ethernet.dst_addr = dst;

        // inform that ipv4 checksum need to be re-calculated.  
        meta.ipv4_checksum_needed = true;
    }

    table ipv4_forward {
        key = { 
            meta.rd           : exact;
            hdr.ipv4.dst_addr : lpm;
        }
        actions = { 
            forwarding_v4;
            forwarding_to_CPU;
            NoAction;
        }
        default_action = NoAction;
        size = 1024;
    }

    action forwarding_v6(bit<9> port, mac_addr_t dst, mac_addr_t src) {

        // set forwarding port
        std_meta.egress_spec = port;

        // update MAC addr
        hdr.ethernet.src_addr = src;
        hdr.ethernet.dst_addr = dst;
    }

    table ipv6_forward {
        key = { 
            meta.rd           : exact;
            hdr.ipv6.dst_addr : lpm;
        }        
        actions = {
            forwarding_v6;
            forwarding_to_CPU;
            NoAction;
        }
        default_action = NoAction;
        size = 1024;
    }

    action get_rd_from_vlan(bit<64> rd) {
        meta.rd = rd;
    }

    table vlan_rd_mapping {
        key = {
            hdr.vlan.id : exact;
        }
        actions = {
            get_rd_from_vlan;
            NoAction;
        }
        default_action = NoAction;
        size = 1024;
    }

    action end() {

        // forward w/ IPv6 dst addr
        meta.ipv6_forward_flag = true;

        // save srv6 func. applied
        meta.srv6_func_applied = END;
    }

    /* TODO: dst port MUST be a set of L3 adjacencies and the one MUST be selected based on Hash.
              By now, the dst port is set only one.
    */
    action end_x(bit<9> port, mac_addr_t dst, mac_addr_t src) {

        // set forwarding port
        std_meta.egress_spec = port;

        // update MAC addr. with action params.
        hdr.ethernet.src_addr = src;
        hdr.ethernet.dst_addr = dst;

        // save srv6 func. applied
        meta.srv6_func_applied = END_X;
    }

    action end_dx2(bit<9> port) {

        // set forwarding port
        std_meta.egress_spec = port;

        // replace inner ethernet header to outer ethernet header
        // this is equal to expose inner ethernet header
        hdr.ethernet.dst_addr = hdr.inner_ether.dst_addr;
        hdr.ethernet.src_addr = hdr.inner_ether.src_addr;
        hdr.ethernet.ether_type = hdr.inner_ether.ether_type;
        hdr.inner_ether.setInvalid();

        // save srv6 func. applied
        meta.srv6_func_applied = END_DX2;
    }

    action end_dx2v(bit<64> rd) {

        // save RD 
        meta.rd = rd;

        // replace inner ethernet header to outer ethernet header
        // this is equal to expose inner ethernet header
        hdr.ethernet.dst_addr = hdr.inner_ether.dst_addr;
        hdr.ethernet.src_addr = hdr.inner_ether.src_addr;
        hdr.ethernet.ether_type = hdr.inner_ether.ether_type;
        hdr.inner_ether.setInvalid();

        // save srv6 func. applied
        meta.srv6_func_applied = END_DX2V;
    }

    action end_dt2u(bit<64> rd) {

        // save RD
        meta.rd = rd;

        // replace inner ethernet header to outer ethernet header
        // this is equal to expose inner ethernet header
        hdr.ethernet.dst_addr = hdr.inner_ether.dst_addr;
        hdr.ethernet.src_addr = hdr.inner_ether.src_addr;
        hdr.ethernet.ether_type = hdr.inner_ether.ether_type;
        hdr.inner_ether.setInvalid();

        // save SRv6 func. applied
        meta.srv6_func_applied = END_DT2U;
    }

    action end_dt2m(bit<16> mcast_grp) {

        // replace inner ethernet header to outer ethernet header
        // this is equal to expose inner ethernet header
        hdr.ethernet.dst_addr = hdr.inner_ether.dst_addr;
        hdr.ethernet.src_addr = hdr.inner_ether.src_addr;
        hdr.ethernet.ether_type = hdr.inner_ether.ether_type;
        hdr.inner_ether.setInvalid();

        // forward to multiple ports associated with "mcast_grp"
        std_meta.mcast_grp = mcast_grp;

        // save SRv6 func. applied
        meta.srv6_func_applied = END_DT2M;
    }

    table srv6_func {
        key = {
            hdr.ipv6.dst_addr : lpm;
        }
        actions = {
            end;
            end_x;
            end_dx2;
            end_dx2v;
            end_dt2u;
            end_dt2m;
            NoAction;
        }
        default_action = NoAction;
        size = 1024;
    }

    action extract_next_sid_0() {
        meta.next_sid = hdr.sid_stack[0].sid;
        meta.psp_flag = true;
    }

    action extract_next_sid_1() {
        meta.next_sid = hdr.sid_stack[1].sid;
    }

    action extract_next_sid_2() {
        meta.next_sid = hdr.sid_stack[2].sid;
    }
    
    table extract_next_sid {
        key = { hdr.srh.seg_left : exact; }
        actions = {
            extract_next_sid_0;
            extract_next_sid_1;
            extract_next_sid_2;
            NoAction;
        }
        const entries = {
            (0x00) : extract_next_sid_0();
            (0x01) : extract_next_sid_1();
            (0x02) : extract_next_sid_2();
        }
        default_action = NoAction;
    }

    action setInvalid_sid_stack_0() {

        // update next header with SRH
        hdr.ipv6.next_hdr = hdr.srh.next_hdr;

        // update length
        hdr.ipv6.payload_len = hdr.ipv6.payload_len - 24;

        // remove SRH, SID stack
        hdr.srh.setInvalid();
        hdr.sid_stack[0].setInvalid();
    }

    action setInvalid_sid_stack_1() {

        // update next header with SRH
        hdr.ipv6.next_hdr = hdr.srh.next_hdr;

        // update length
        hdr.ipv6.payload_len = hdr.ipv6.payload_len - 40;

        // remove SRH, SID stack
        hdr.srh.setInvalid();
        hdr.sid_stack[0].setInvalid();
        hdr.sid_stack[1].setInvalid();
    }

    action setInvalid_sid_stack_2() {

         
        // update next header with SRH
        hdr.ipv6.next_hdr = hdr.srh.next_hdr;

        // update length 
        hdr.ipv6.payload_len = hdr.ipv6.payload_len - 56;

        // remove SRH, SID stack
        hdr.srh.setInvalid();
        hdr.sid_stack[0].setInvalid();
        hdr.sid_stack[1].setInvalid();
        hdr.sid_stack[2].setInvalid();
    }

    table setInvalid_srh_0 {
        key = { hdr.srh.last_entry : exact; }
        actions = {
            setInvalid_sid_stack_0;
            setInvalid_sid_stack_1;
            setInvalid_sid_stack_2;
            NoAction;
        }
        const entries = {
            (0x00) : setInvalid_sid_stack_0();
            (0x01) : setInvalid_sid_stack_1();
            (0x02) : setInvalid_sid_stack_2();
        }
        default_action = NoAction;
    }

    table setInvalid_srh_1 {
        key = { hdr.srh.last_entry : exact; }
        actions = {
            setInvalid_sid_stack_0;
            setInvalid_sid_stack_1;
            setInvalid_sid_stack_2;
            NoAction;
        }
        const entries = {
            (0x00) : setInvalid_sid_stack_0();
            (0x01) : setInvalid_sid_stack_1();
            (0x02) : setInvalid_sid_stack_2();
        }
        default_action = NoAction;
    }

    action send_ICMP_error(bit<8> code) {
        /* TODO: implement action to send ICMP error to hdr.ipv4/v6.src_addr with error code "code" */
    }

	apply {

        if (!l2_check.apply().hit) {
            // NOTE: (ingress_port, VLAN ID, dst. MAC) of received packet is NOT allowed. 
            //       this packet SHOULD be dropped.
        }

        // check if in-coming packet only needs switching or not.
        vlan_rd_mapping.apply()
        if (!l2_forward.) {

        } else if (srv6_encaps_func.apply().hit) {
            if (hdr.vlan.isValid()) {
                hdr.inner_vlan.setValid();
                hdr.inner_vlan.pcp = hdr.vlan.pcp;
                hdr.inner_vlan.cfi = hdr.vlan.cfi;
                hdr.inner_vlan.id = hdr.vlan.id;
                hdr.inner_vlan.ether_type = hdr.vlan.ether_type;

                hdr.ipv6.payload_len = hdr.ipv6.payload_len + 4;

                hdr.vlan.setInvalid();
            }
            if (hdr.ipv4.isValid()) {
                hdr.inner_ipv4.setValid();
                hdr.inner_ipv4.version = hdr.ipv4.version;
                hdr.inner_ipv4.ihl = hdr.ipv4.ihl;
                hdr.inner_ipv4.diffserv = hdr.ipv4.diffserv;
                hdr.inner_ipv4.total_len = hdr.ipv4.total_len;
                hdr.inner_ipv4.identification = hdr.ipv4.identification;
                hdr.inner_ipv4.flags = hdr.ipv4.flags;
                hdr.inner_ipv4.flag_offset = hdr.ipv4.flag_offset;
                hdr.inner_ipv4.ttl = hdr.ipv4.ttl;
                hdr.inner_ipv4.protocol = hdr.ipv4.protocol;
                hdr.inner_ipv4.hdr_checksum = hdr.ipv4.hdr_checksum;
                hdr.inner_ipv4.src_addr = hdr.ipv4.src_addr;
                hdr.inner_ipv4.dst_addr = hdr.ipv4.dst_addr;

                hdr.ipv6.payload_len = hdr.ipv6.payload_len + 20;
                hdr.ipv6.traffic_class = hdr.ipv4.diffserv;

                hdr.ipv4.setInvalid();
            }
        }

		if (hdr.ipv6.isValid()) {

            // apply SRv6 functions
            if (srv6_func.apply().hit) {
                if (hdr.srh.isValid()) {
                    if (hdr.srh.seg_left == 0) {

                        // final end-point function w/ SRH MUST be applied
                        if (meta.srv6_func_applied == END_DX2 && hdr.srh.next_hdr == PROTOCOL_ETHER) {
                        
                            // replace inner vlan tag to outer vlan tag if necessary
                            // this is equal to expose inner vlan tag
                            if (hdr.inner_vlan.isValid()) {
                                hdr.vlan.setValid();
                                hdr.vlan.pcp = hdr.inner_vlan.pcp;
                                hdr.vlan.cfi = hdr.inner_vlan.cfi;
                                hdr.vlan.id = hdr.inner_vlan.id;
                                hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                                hdr.inner_vlan.setInvalid();
                            }
                        
                        } else if (meta.srv6_func_applied == END_DX2V && hdr.srh.next_hdr == PROTOCOL_ETHER) {

                            // replace inner vlan tag to outer vlan tag if necessary
                            // this is equal to expose inner vlan tag
                            if (hdr.inner_vlan.isValid()) {
                                hdr.vlan.setValid();
                                hdr.vlan.pcp = hdr.inner_vlan.pcp;
                                hdr.vlan.cfi = hdr.inner_vlan.cfi;
                                hdr.vlan.id = hdr.inner_vlan.id;
                                hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                                hdr.inner_vlan.setInvalid();
                            } else {
                                /* TODO: inner_vlan MUST be parsed because this action requires inner vlan ID.
                                         if hdr.inner_vlan.isValid() is false packet SHOULD be dropped and ICMP error be sent.
                                */
                            }

                            // forwarding w/ inner VLAN ID
                            meta.vlan_id = hdr.inner_vlan.id;
                            meta.vlan_forward_flag = true;

                        } else if (meta.srv6_func_applied == END_DT2U && hdr.srh.next_hdr == PROTOCOL_ETHER) {
                        
                            // replace inner vlan tag to outer vlan tag if necessary
                            // this is equal to expose inner vlan tag
                            if (hdr.inner_vlan.isValid()) {
                                hdr.vlan.setValid();
                                hdr.vlan.pcp = hdr.inner_vlan.pcp;
                                hdr.vlan.cfi = hdr.inner_vlan.cfi;
                                hdr.vlan.id = hdr.inner_vlan.id;
                                hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                                hdr.inner_vlan.setInvalid();
                            }

                            // forwarding w/ exposed MAC dst. addr
                            meta.ethernet_dst_addr = hdr.inner_ether.dst_addr;
                            meta.l2_forward_flag = true;

                            /* TODO: learn src. MAC in table w/ RD (refer to RFC8986 4.11)*/

                        } else if (meta.srv6_func_applied == END_DT2M && hdr.srh.next_hdr == PROTOCOL_ETHER) {

                            // remove outer ether header (expose inner ether header)
                            if (hdr.inner_vlan.isValid()) {
                                hdr.vlan.setValid();
                                hdr.vlan.pcp = hdr.inner_vlan.pcp;
                                hdr.vlan.cfi = hdr.inner_vlan.cfi;
                                hdr.vlan.id = hdr.inner_vlan.id;
                                hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                                hdr.inner_vlan.setInvalid();
                            }

                            /* TODO: learn src. MAC in table w/ RD (refer to RFC8986 4.11)*/

                        } else {
                            // SRv6 func. for final end-point is applied, but the inner header is invalid.
                            // or SRv6 func. for intermediate end-point is applied with seg_left == 0.
                            meta.upper_layer_proc_flag = true;
                        }

                        // upper layer processing (if any)
                        if (meta.upper_layer_proc_flag) {
                            if (hdr.srh.next_hdr == 0xff) {
                                // processing for upper layer (if any)
                                // 0xff is meaningless value here ...
                            } else {
                                send_ICMP_error(4);
                                mark_to_drop(std_meta);
                            }
                        }

                        // remove IPv6, SRH, SID stack
                        setInvalid_srh_0.apply();
                        hdr.ipv6.setInvalid();

                    } else {

                        // intermediate end-point functions MUST be applied 
                        if (hdr.ipv6.hop_limit <= 1) {
                            send_ICMP_error(0);
                            mark_to_drop(std_meta);
                        } else {
                            hdr.ipv6.hop_limit = hdr.ipv6.hop_limit - 1;
                        }
                        hdr.srh.seg_left = hdr.srh.seg_left - 1;
                        extract_next_sid.apply();
                        if (meta.srv6_func_applied == END) {
                            // update IPv6 dst addr with active SID
                            hdr.ipv6.dst_addr = meta.next_sid;
                        } else if (meta.srv6_func_applied == END_X) {
                            // update IPv6 dst addr with active SID
                            hdr.ipv6.dst_addr = meta.next_sid;
                        } else {
                            // SRv6 func. for final end point is applied with seg_left != 0
                            send_ICMP_error(0);
                            mark_to_drop(std_meta);
                        }

                        // PSP (if any)
                        if (meta.psp_flag) {
                            setInvalid_srh_1.apply();
                        }
                    }
                } else {

                    // final end-point function w/o SRH MUST be applied
                    if (meta.srv6_func_applied == END_DX2 && hdr.ipv6.next_hdr == PROTOCOL_ETHER) {
                        
                        // replace inner vlan tag to outer vlan tag if necessary
                        // this is equal to expose inner vlan tag
                        if (hdr.inner_vlan.isValid()) {
                            hdr.vlan.setValid();
                            hdr.vlan.pcp = hdr.inner_vlan.pcp;
                            hdr.vlan.cfi = hdr.inner_vlan.cfi;
                            hdr.vlan.id = hdr.inner_vlan.id;
                            hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                            hdr.inner_vlan.setInvalid();
                        }
                        
                    } else if (meta.srv6_func_applied == END_DX2V && hdr.ipv6.next_hdr == PROTOCOL_ETHER) {

                        // replace inner vlan tag to outer vlan tag if necessary
                        // this is equal to expose inner vlan tag
                        if (hdr.inner_vlan.isValid()) {
                            hdr.vlan.setValid();
                            hdr.vlan.pcp = hdr.inner_vlan.pcp;
                            hdr.vlan.cfi = hdr.inner_vlan.cfi;
                            hdr.vlan.id = hdr.inner_vlan.id;
                            hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                            hdr.inner_vlan.setInvalid();
                        } else {
                            /* TODO: inner_vlan MUST be parsed because this action requires inner vlan ID.
                                     if hdr.inner_vlan.isValid() is false packet SHOULD be dropped and ICMP error be sent.
                            */
                        }

                        // forwarding w/ inner VLAN ID
                        meta.vlan_id = hdr.inner_vlan.id;
                        meta.vlan_forward_flag = true;

                    } else if (meta.srv6_func_applied == END_DT2U && hdr.ipv6.next_hdr == PROTOCOL_ETHER) {
                        
                        // replace inner vlan tag to outer vlan tag if necessary
                        // this is equal to expose inner vlan tag
                        if (hdr.inner_vlan.isValid()) {
                            hdr.vlan.setValid();
                            hdr.vlan.pcp = hdr.inner_vlan.pcp;
                            hdr.vlan.cfi = hdr.inner_vlan.cfi;
                            hdr.vlan.id = hdr.inner_vlan.id;
                            hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                            hdr.inner_vlan.setInvalid();
                        }

                        // forwarding w/ exposed MAC dst. addr
                        meta.ethernet_dst_addr = hdr.inner_ether.dst_addr;
                        meta.l2_forward_flag = true;

                        /* TODO: learn src. MAC in table w/ RD (refer to RFC8986 4.11)*/

                    } else if (meta.srv6_func_applied == END_DT2M && hdr.ipv6.next_hdr == PROTOCOL_ETHER) {

                        // replace inner vlan tag to outer vlan tag if necessary
                        // this is equal to expose inner vlan tag
                        if (hdr.inner_vlan.isValid()) {
                            hdr.vlan.setValid();
                            hdr.vlan.pcp = hdr.inner_vlan.pcp;
                            hdr.vlan.cfi = hdr.inner_vlan.cfi;
                            hdr.vlan.id = hdr.inner_vlan.id;
                            hdr.vlan.ether_type = hdr.inner_vlan.ether_type;
                            hdr.inner_vlan.setInvalid();
                        }

                        /* TODO: learn src. MAC in table w/ RD (refer to RFC8986 4.11)*/

                    } else {
                        // SRv6 func. for final end-point is applied, but the inner header is invalid.
                        // or SRv6 func. for intermediate end-point is applied with seg_left == 0.
                        meta.upper_layer_proc_flag = true;
                    }

                    // upper layer processing (if any)
                    if (meta.upper_layer_proc_flag) {
                        if (hdr.ipv6.next_hdr == 0xff) {
                            // processing for upper layer (if any)
                            // 0xff is meaningless value here ...
                        } else {
                            send_ICMP_error(4);
                            mark_to_drop(std_meta);
                        }
                    }

                    // remove IPv6 header only
                    // note that SRH and SID stack has been already removed
                    hdr.ipv6.setInvalid();
                }
            } else {
                if (hdr.ipv6.hop_limit <= 1) {
                    send_ICMP_error(0);
                    mark_to_drop(std_meta);
                } else {
                    hdr.ipv6.hop_limit = hdr.ipv6.hop_limit - 1;
                }
                meta.ipv6_forward_flag = true;
            }
        } else if (hdr.ipv4.isValid()) {
            if (hdr.ipv4.ttl <= 1) {
                send_ICMP_error(0);
                mark_to_drop(std_meta);
            } else {
                hdr.ipv4.ttl = hdr.ipv4.ttl - 1;
            }
            meta.ipv4_forward_flag = true;
        } else if (hdr.ethernet.isValid()) {
            vlan_rd_mapping.apply();
            meta.l2_forward_flag = true;
        }

        // forwarding actions if necessary
        if (meta.ipv4_forward_flag) {
            ipv4_forward.apply();
        } else if (meta.ipv6_forward_flag) {
            ipv6_forward.apply();
        } else if (meta.l2_forward_flag) {
            l2_forward.apply();
        } else if (meta.vlan_forward_flag) {
            vlan_forward.apply();
        }
    }
}

/********** EGRESS **********/
control SwitchEgress(inout header_t hdr,
                	 inout metadata_t meta,
                	 inout standard_metadata_t std_meta)
{
    apply {}
}

/*********** COMPUTE CHECKSUM **********/
control SwitchComputeChecksum(inout header_t hdr, inout metadata_t meta)
{
	apply {
        update_checksum(
            meta.ipv4_checksum_needed,
            { hdr.ipv4.version,
              hdr.ipv4.ihl,
              hdr.ipv4.diffserv,
              hdr.ipv4.total_len,
              hdr.ipv4.identification,
              hdr.ipv4.flags,
              hdr.ipv4.flag_offset,
              hdr.ipv4.ttl,
              hdr.ipv4.protocol,
              hdr.ipv4.src_addr,
              hdr.ipv4.dst_addr},
            hdr.ipv4.hdr_checksum,
            HashAlgorithm.csum16);  
    }
}

/*********** DEPARSER **********/
control SwitchDeparser(packet_out pkt, in header_t hdr)
{
    apply { 
        pkt.emit(hdr.packetIn);
        pkt.emit(hdr.ethernet);
        pkt.emit(hdr.vlan);
        pkt.emit(hdr.ipv4);
        pkt.emit(hdr.ipv6);
        pkt.emit(hdr.srh);
        pkt.emit(hdr.sid_stack);
        pkt.emit(hdr.inner_ether);
        pkt.emit(hdr.inner_vlan);
        pkt.emit(hdr.inner_ipv4);
    }
}

/*********** PIPELINE ***********/
V1Switch(
SwitchParser(),
SwitchVerifyChecksum(),
SwitchIngress(),
SwitchEgress(),
SwitchComputeChecksum(),
SwitchDeparser()
) main;