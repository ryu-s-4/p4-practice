package myutils

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"

	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

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
	/* TODO */
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
	/* TODO */
	dammy int
}

// DigestEntryHelper is helper for DigestEntry.
type DigestEntryHelper struct {
	/* TODO */
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
		err := fmt.Errorf("cannot find table instance")
		return nil, err
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
			err := fmt.Errorf("cannot find match field instance")
			return nil, err
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
			/* TODO */
			/*
				fm := &v1.FieldMatch{
					FieldMatchType: &v1.FieldMatch_Ternary_{},
				}
			*/

		case "RANGE":
			/* TODO */
			/*
				fm := &v1.FieldMatch{
					FieldMatchType: &v1.FieldMatch_Range_{},
				}
			*/

		default:
			/* TODO */
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
				p, err := GetParam(value, param.Bitwidth)
				if err != nil {
					return nil, err
				}
				action_param := &v1.Action_Param{
					ParamId: param.Id,
					Value:   p,
				}
				action_params = append(action_params, action_param)
				flag = true
				break
			}
		}
		if flag == false {
			err := fmt.Errorf("cannot find action parameters")
			return nil, err
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
func GetParam(value interface{}, width int32) ([]byte, error) {

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
		binary.BigEndian.PutUint64(param, uint64(value.(float64)))

	case string:
		if width == 48 {
			var err error
			param, err = net.ParseMAC(value.(string))
			if err != nil {
				// Error 処理
				return nil, err
			}
		} else {
			param = net.ParseIP(value.(string))
			if param == nil {
				err := fmt.Errorf("cannot parse %s", value.(string))
				return nil, err
			}
		}

	default:
		param = make([]byte, upper)
	}

	return param[(len(param) - upper):], nil
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
		err := fmt.Errorf("cannot find counter instance")
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

// GetCounterSpec_Unit gets the unit of "counter" instance.
func GetCounterSpec_Unit(counter string, p config_v1.P4Info) config_v1.CounterSpec, error {

	// "counter" を探索
	var flag bool
	var cnt config_v1.Counter

	flag = false
	for _, c := range p.Counters {
		if c.Preamble.Name == counter {
			cnt = c
			flag = true
			break
		}
	}
	if flag == false {
		err := fmt.Errorf("cannot find counter instance")
		return nil, err
	}

	// CounterSpec_Unit を return 
	return c.Spec.Unit, nil
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
