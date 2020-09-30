// package helper
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// P4InfoHelper is helper for p4info.
type P4InfoHelper struct {
	p4info    *config_v1.P4Info
	enthelper EntryHelper
}

// ConfigHelper is helper for config setup.
type ConfigHelper struct {
	Target    string `json:"target"`
	P4info    string `json:"p4info"`
	BMv2_json string `json:"bmv2_json"`
}

// EntryHelper is helper for Entry
type EntryHelper struct {
	TableEntry          []*TableEntryHelper          `json:"table_entries"`
	MulticastGroupEntry []*MulticastGroupEntryHelper `json:"multicast_group_entries"`
}

// TableEntryHelper is helper for TableEntry.
type TableEntryHelper struct {
	Table         string                 `json:"table"`
	Match         map[string]interface{} `json:"match"`
	Action_Name   string                 `json:"action_name"`
	Action_Params map[string]interface{} `json:"action_params"`
}

/*
// GetParams gets action parameters corresponding to key in []byte.
func (t *TableEntryHelper) GetParams(key string) ([]byte, error) {
	// key で action parameter を取得し []byte に変換して値を返す
	// MAC アドレス，IPv4 アドレス，IPv6 アドレスのパースにも対応（したい）
}
*/

type MulticastGroupEntryHelper struct {
	Multicast_Group_ID uint32           `json:"multicast_group_id"`
	Replicas           []*ReplicaHelper `json:"replicas"`
}

type ReplicaHelper struct {
	Egress_port uint32 `json:"egress_port"`
	Instance    uint32 `json:"instance"`
}

// BuildTableEntry creates TableEntry object.
func (h *TableEntryHelper) BuildTableEntry(p config_v1.P4Info) (*v1.TableEntry, error) {

	// Get table
	var table *config_v1.Table
	for _, t := range p.Tables {
		if t.Preamble.Name == h.Table {
			table = t
			break
		}
	}
	if table == nil {
		// Error 処理
	}

	// Get table id
	tableid := table.Preamble.Id

	// Get fieldmatch
	var fieldmatch []*v1.FieldMatch
	for key, value := range h.Match {

		flag := false
		match := &config_v1.MatchField{}
		for _, m := range table.MatchFields {
			if m.Name == key {
				match = m
				flag = true
				break
			}
		}
		if flag != true {
			// Error 処理
		}

		switch match.GetMatchType().String() {

		case "EXACT":
			v := value.([]byte)

			fm := &v1.FieldMatch{
				FieldId: match.Id,
				FieldMatchType: &v1.FieldMatch_Exact_{
					Exact: &v1.FieldMatch_Exact{
						Value: v,
					},
				},
			}
			fieldmatch = append(fieldmatch, fm)

		case "LPM":
			v, err := GetAddr(value.([]interface{})[0].(string))
			if err != nil {
				// Error 処理
			}
			prefix := value.([]interface{})[1].(int32)

			fm := &v1.FieldMatch{
				FieldId: match.Id,
				FieldMatchType: &v1.FieldMatch_Lpm{
					Lpm: &v1.FieldMatch_LPM{
						Value:     v,
						PrefixLen: prefix,
					},
				},
			}
			fieldmatch = append(fieldmatch, fm)

		case "TERNARY":
			match := v1.FieldMatch_Ternary_{}
		case "RANGE":
			match := v1.FieldMatch_Range_{}
		default:
			match := v1.FieldMatch_Other{}
		}

	}

	// Get action
}

func GetAddr(addr string) ([]byte, error) {

	// MAC Addr.
	return net.ParseMAC(addr)

	// IPv4 Addr.

	// IPv6 Addr.
}

/*
func (h *EntryHelper) BuildMulticastGroupEntry() (*v1.MulticastGroupEntry, error) {
}
*/

func main() {

	enthelp := EntryHelper{}
	confhelp := ConfigHelper{}

	runtime, err := ioutil.ReadFile("./runtime.json")
	if err != nil {
		log.Fatal("Error.", err)
	}

	if err := json.Unmarshal(runtime, &confhelp); err != nil {
		log.Fatal("Error.", err)
	}

	if err := json.Unmarshal(runtime, &enthelp); err != nil {
		log.Fatal("Error.", err)
	}

	// fmt.Println(string(runtime))
	// fmt.Println(confhelp.P4info)
	switch enthelp.TableEntry[0].Match["hdr.ipv4.dstAddr"].(type) {
	case []interface{}:
		fmt.Println("OK")
	default:
		fmt.Println("NG")
	}
}
