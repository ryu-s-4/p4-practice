package myutils

import (
	"encoding/binary"
	"log"
	"net"
	"fmt"

	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// ConfigHelper is helper for config setup.
/*
type ConfigHelper struct {
	Target    string `json:"target"`
	P4info    string `json:"p4info"`
	BMv2_json string `json:"bmv2_json"`
}
*/

// EntryHelper is helper for Entry
type EntryHelper struct {
	ExternEntries         []*ExternEntryHelper         `json:"extern_entries"`
	TableEntries          []*TableEntryHelper          `json:"table_entries"`
	MeterEntries          []*MeterEntryHelper          `json:"meter_entries"`
	CounterEntries        []*CounterEntryHelper        `json:"counter_entries"`
	MulticastGroupEntries []*MulticastGroupEntryHelper `json:"multicast_group_entries"`
	RegisterEntries       []*RegisterEntryHelper       `json:"register_entries"`
	DigestEntries         []*DigestEntryHelper         `json:digest_entries"`
}

// ExternEntryHelper is helper for ExternEntry.
type ExternEntryHelper struct {
	dammy int
}

// TableEntryHelper is helper for TableEntry.
type TableEntryHelper struct {
	Table         string                 `json:"table"`
	Match         map[string]interface{} `json:"match"`
	Action_Name   string                 `json:"action_name"`
	Action_Params map[string]interface{} `json:"action_params"`
}

// MeterEntryHelper is helper for MeterEntry.
type MeterEntryHelper struct {
	dammy int
}

// CounterEntryHelper is helper for CounterEntry.
type CounterEntryHelper struct {
	Counter string `json:"counter"`
	Index   int64  `json:index"`
}

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

// RegisterEntryHepler is hepler for RegisterEntry.
type RegisterEntryHelper struct {
	dammy int
}

// DigestEntryHelper is helper for DigestEntry.
type DigestEntryHelper struct {
	dammy int
}

// BuildTableEntry creates TableEntry in the form of Entity_TableEntry.
func BuildTableEntry(h *TableEntryHelper, p config_v1.P4Info) (*v1.Entity_TableEntry, error) {

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
		log.Fatal("Error: Not Found Table.")
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
			log.Fatal("Error: Not Found MatchField.")
		}

		switch match.GetMatchType().String() {

		case "EXACT":

			v := GetParam(value, match.Bitwidth)
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

			var v []byte
			v = net.ParseIP(value.([]interface{})[0].(string))
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
			/*
				fm := &v1.FieldMatch{
					FieldMatchType: &v1.FieldMatch_Ternary_{},
				}
			*/

		case "RANGE":
			/*
				fm := &v1.FieldMatch{
					FieldMatchType: &v1.FieldMatch_Range_{},
				}
			*/

		default:
			/*
				fm := &v1.FieldMatch{
					FieldMatchType: &v1.FieldMatch_Other{},
				}
			*/
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
		log.Fatal("Error: Not Found Action.")
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
					Value:   GetParam(value, param.Bitwidth), // TODO: MAC アドレス等の場合に net.ParseMAC 必要
				}
				action_params = append(action_params, action_param)
				flag = true
				break
			}
		}
		if flag == false {
			log.Fatal("Error: Not Found Action Params.")
		}
	}

	// return TableEntry
	entityTableEntry := &v1.Entity_TableEntry{
		TableEntry: &v1.TableEntry{
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
		},
	}

	return entityTableEntry, nil
}

// GetParam gets action parameter in []byte
func GetParam(value interface{}, width int32) []byte {

	var upper int
	if width%8 == 0 {
		upper = int(width / 8)
	} else {
		upper = int(width/8) + 1
	}

	var param []byte
	switch value.(type) {

	case float64:
		param = make([]byte, 8)
		// binary.LittleEndian.PutUint64(param, uint64(value.(float64)))
		binary.BigEndian.PutUint64(param, uint64(value.(float64)))

	case string:
		if width == 48 {
			var err error
			param, err = net.ParseMAC(value.(string))
			if err != nil {
				// Error 処理
				log.Fatal("ERROR: ParseMAC", err)
			}
		} else {
			param = net.ParseIP(value.(string))
			if param == nil {
				// Error 処理
			}
		}

	default:
		param = make([]byte, upper)
	}

	/*
		var a int
		fmt.Println("parameter: ", param)         // DEBUG
		fmt.Println("upper    : ", upper)         // DEBUG
		fmt.Println("param    : ", param[:upper]) // DEBUG
		fmt.Scan(&a)
	*/
	// return param[:upper] // FOR Little Endian
	return param[(len(param) - upper):]
}

// BuildMulticastGroupEntry creates MulticastGroupEntry in the form of Entity_PacketRelicationEngineEntry.
func BuildMulticastGroupEntry(h *MulticastGroupEntryHelper) (*v1.Entity_PacketReplicationEngineEntry, error) {

	// Get multicast group id
	groupid := h.Multicast_Group_ID

	// Get Replicas
	replicas := make([]*v1.Replica, 0)
	for _, r := range h.Replicas {
		replicas = append(replicas, &v1.Replica{EgressPort: r.Egress_port, Instance: r.Instance})
	}

	// Return Entity_PacketReplicationEngineEntry
	entity_PacketReplicationEngineEntry := &v1.Entity_PacketReplicationEngineEntry{
		PacketReplicationEngineEntry: &v1.PacketReplicationEngineEntry{
			Type: &v1.PacketReplicationEngineEntry_MulticastGroupEntry{
				MulticastGroupEntry: &v1.MulticastGroupEntry{
					MulticastGroupId: groupid,
					Replicas:         replicas,
				},
			},
		},
	}

	return entity_PacketReplicationEngineEntry, nil
}

func BuildCounterEntry(h *CounterEntryHelper, p config_v1.P4Info) (*v1.Entity_CounterEntry, error) {

	// get counter
	var flag bool
	var counter *config_v1.Counter

	flag = false
	for _, c := range p.Counters {
		if (c.Preamble.Name == h.Counter) || (c.Preamble.Alias == h.Counter) {
			counter = c
			flag = true
			break
		}
	}
	if flag == false {
		// Error 処理
		err := fmt.Errorf("cannot find counter instance.")
		return nil, err
	}

	// return Counter
	entity_counterentry := &v1.Entity_CounterEntry{
		CounterEntry: &v1.CounterEntry{
			CounterId: counter.Preamble.Id,
			Index: &v1.Index{
				Index: h.Index,
			},
		},
	}

	return entity_counterentry, nil
}

// NewUpdate creates new Update instance.
func NewUpdate(updateType string, entity *v1.Entity) (*v1.Update, error) {

	switch updateType {
	case "INSERT":
		update := v1.Update{
			Type:   v1.Update_INSERT,
			Entity: entity}
		return &update, nil

	case "MODIFY":
		update := v1.Update{
			Type:   v1.Update_MODIFY,
			Entity: entity}
		return &update, nil

	case "DELETE":
		update := v1.Update{
			Type:   v1.Update_DELETE,
			Entity: entity}
		return &update, nil

	default:
		update := v1.Update{
			Type:   v1.Update_UNSPECIFIED,
			Entity: entity}
		return &update, nil
	}
}

/*
func main() {

	runtime_path := "../runtime.json"
	p4info_path := "../p4info.txt"

	runtime, err := ioutil.ReadFile(runtime_path)
	if err != nil {
		// ReadFile Error
		log.Fatal("Error")
	}

	p4info_row, err := ioutil.ReadFile(p4info_path)
	if err != nil {
		// ReadFile Error
		log.Fatal("Error")
	}

	var entryhelper EntryHelper
	if err := json.Unmarshal(runtime, &entryhelper); err != nil {
		// Error 処理
		log.Fatal("Error")
	}

	p4info := config_v1.P4Info{}
	if err := proto.UnmarshalText(string(p4info_row), &p4info); err != nil {
		// Error 処理
		log.Fatal("Error")
	}

	var tableentries []*v1.Entity_TableEntry
	for _, tableenthelper := range entryhelper.TableEntries {
		tableentry, err := BuildTableEntry(tableenthelper, p4info)
		if err != nil {
			// Error 処理
			log.Fatal("Error Build.")
		}
		tableentries = append(tableentries, tableentry)
		fmt.Println(tableentry) // For DEBUG
	}

	var multicastgroupenties []*v1.Entity_PacketReplicationEngineEntry
	for _, multicastgroupenthelper := range entryhelper.MulticastGroupEntries {
		multicastgroupentry, err := BuildMulticastGroupEntry(multicastgroupenthelper)
		if err != nil {
			// Error 処理
		}
		multicastgroupenties = append(multicastgroupenties, multicastgroupentry)
		fmt.Println(multicastgroupentry) // For DEBUG
	}

	var counterentries []*v1.Entity_CounterEntry
	for _, counterentryhelper := range entryhelper.CounterEntries {
		counterentry, err := BuildCounterEntry(counterentryhelper, p4info)
		if err != nil {
			// Error 処理
		}
		counterentries = append(counterentries, counterentry)
		fmt.Println(counterentry) // For DEBUG
	}
}
*/
