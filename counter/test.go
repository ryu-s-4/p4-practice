package main

import (
	"fmt"
	"io/ioutil"
	"os"

	v1 "github.com/p4lang/p4runtime/go/p4/v1"
)

func main() {
	var entry v1.TableEntry

	filename := "./p4rt_tableentries.json"
	conf, err := ioutil.ReadFile(filename)
	if err != nil {
		// Error 処理
		os.Exit(1)
	}

	entry.XXX_Unmarshal(conf)
	//json.Unmarshal(conf, &entry)
	fmt.Println(entry)
}

/*
func main() {

	tableID := uint32(99)                             // TODO: replace with table id from p4info file.
	fieldID := uint32(99)                             // TODO: replace with field id from p4info file.
	vlanID := []byte{uint8(63)}                       // TODO: replace with vlan-id what you want.
	macAddr, err := net.ParseMAC("00:11:22:33:44:55") // TODO: replace with mac addr. what you want.
	if err != nil {
		// Error 処理
	}
	actionID := uint32(99)                    // TODO: replace with action id from p4info file.
	paramID := uint32(99)                     // TODO: replace with param id from p4info file.
	portNum := []byte{uint8(128), uint8(128)} // TODO: replace with port num. what you want.

	tableEntry := v1.TableEntry{
		TableId: tableID,
		Match: []*v1.FieldMatch{
			{
				FieldId: fieldID,
				FieldMatchType: &v1.FieldMatch_Exact_{
					Exact: &v1.FieldMatch_Exact{
						Value: vlanID,
					},
				},
			},
			{
				FieldId: fieldID,
				FieldMatchType: &v1.FieldMatch_Exact_{
					Exact: &v1.FieldMatch_Exact{
						Value: macAddr,
					},
				},
			},
		},
		Action: &v1.TableAction{
			Type: &v1.TableAction_Action{
				Action: &v1.Action{
					ActionId: actionID,
					Params: []*v1.Action_Param{
						{
							ParamId: paramID,
							Value:   portNum,
						},
					},
				},
			},
		},
	}

	action := tableEntry.Action.GetAction()
	val := action.Params[0].Value
	fmt.Println(binary.BigEndian.Uint16(val))

}
*/
