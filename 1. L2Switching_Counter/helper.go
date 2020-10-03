// package helper
package main

import (
	"encoding/json"
	"io/ioutil"
	"net"

	"github.com/golang/protobuf/proto"
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
	TableEntries          []*TableEntryHelper          `json:"table_entries"`
	MulticastGroupEntries []*MulticastGroupEntryHelper `json:"multicast_group_entries"`
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

// MulticastGroupEntryHelper ...
type MulticastGroupEntryHelper struct {
	Multicast_Group_ID uint32           `json:"multicast_group_id"`
	Replicas           []*ReplicaHelper `json:"replicas"`
}

// ReplicaHelper ...
type ReplicaHelper struct {
	Egress_port uint32 `json:"egress_port"`
	Instance    uint32 `json:"instance"`
}

// BuildTableEntry creates TableEntry object.
func (h *TableEntryHelper) BuildTableEntry(p config_v1.P4Info) (*v1.TableEntry, error) {

	var flag bool

	// Get table
	// TODO: 事前に table 名から Table 構造体を逆引きする map 変数を作成
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

		flag = false
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
			v := net.ParseIP(value.([]interface{})[0].(string))
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
			fm := &v1.FieldMatch{
				FieldMatchType: &v1.FieldMatch_Ternary_{},
			}

		case "RANGE":
			fm := &v1.FieldMatch{
				FieldMatchType: &v1.FieldMatch_Range_{},
			}

		default:
			fm := &v1.FieldMatch{
				FieldMatchType: &v1.FieldMatch_Other{},
			}
		}

	}

	// Get action
	// TODO: 事前に action 名から Action 構造体を逆引きする map 変数を作成
	var action *config_v1.Action

	for _, a := range p.Actions {
		if a.Preamble.Name == h.Action_Name {
			action = a
			break
		}
	}
	if action == nil {
		// Error 処理（Not Found)
	}

	// Get action id
	actionid := action.Preamble.Id

	// Get action parameters
	var action_params []*v1.Action_Param

	for _, param := range action.Params {
		flag = false
		for key, value := range h.Action_Params {
			if key == param.Name {
				action_param := &v1.Action_Param{
					ParamId: param.Id,
					Value:   value.([]byte), // TODO: MAC アドレス等の場合に net.ParseMAC 必要
				}
				action_params = append(action_params, action_param)
			}
		}
		if flag == false {
			// Error 処理 (Not Found)
		}
	}

	// return TableEntry
	tableentry := &v1.TableEntry{
		TableId: tableid,
		Match:   fieldmatch,
		Action: &v1.TableAction{
			Type: &v1.TableAction_Action{
				Action: &v1.Action{
					ActionId: actionid,
					Params:   action_params,
				},
			},
		},
	}
	return tableentry, nil
}

/*
func (h *EntryHelper) BuildMulticastGroupEntry() (*v1.MulticastGroupEntry, error) {
}
*/

func main() {

	runtime_path := "./runtime.json"
	p4info_path := "./p4info.txt"

	runtime, err := ioutil.ReadFile(runtime_path)
	if err != nil {
		// ReadFile Error
	}
	p4info_row, err := ioutil.ReadFile(p4info_path)
	if err != nil {
		// ReadFile Error
	}

	entryhelper := EntryHelper{}
	if err := json.Unmarshal(runtime, &entryhelper); err != nil {
		// Error 処理
	}

	p4info := config_v1.P4Info{}
	if err := proto.UnmarshalText(string(p4info_row), &p4info); err != nil {
		// Error 処理
	}

	for _, tableenthelper := range entryhelper.TableEntries {
		tableentry, err := tableenthelper.BuildTableEntry(p4info)
		if err != nil {
			// Error 処理
		}
	}
}
