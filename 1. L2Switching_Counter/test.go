package main

/*
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
*/

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

/*
func main() {

	var vlanID []byte
	var portNum []byte

	params := make([]byte, 0)

	vlanID = append(vlanID, uint8(0), uint8(1))
	macAddr, _ := net.ParseMAC("ff:ff:ff:ff:ff:ff")
	portNum = append(portNum, uint8(0), uint8(1))

	params = append(params, vlanID...)
	params = append(params, macAddr...)
	params = append(params, portNum...)

	vlan := params[0:2]
	mac := params[2:8]
	port := params[8:]

	fmt.Println(vlan)
	fmt.Println(mac)
	fmt.Println(port)
}
*/

/*
func main() {
	var groupID []byte

	groupID = make([]byte, 2)
	binary.BigEndian.PutUint16(groupID, uint16(10))
	fmt.Println(groupID)

	groupID = make([]byte, 4)
	binary.BigEndian.PutUint32(groupID, uint32(99))
	fmt.Println(groupID)
}
*/

/*
func main() {

	params := make([]byte, 0)
	groupID := make([]byte, 4)
	replica := make([]byte, 8)

	binary.BigEndian.PutUint32(groupID, uint32(1))
	params = append(params, groupID...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(0))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(1))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(2))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	binary.BigEndian.PutUint32(replica[0:4], uint32(3))
	binary.BigEndian.PutUint32(replica[4:8], uint32(1))
	params = append(params, replica...)

	var replicaTest []*v1.Replica
	rep := make([]byte, 8)

	for i := 0; (12 + 8*i) <= len(params); i++ {
		rep = params[(4 + 8*i):(12 + 8*i)]
		replicaTest = append(replicaTest, &v1.Replica{EgressPort: binary.BigEndian.Uint32(rep[0:4]), Instance: binary.BigEndian.Uint32(rep[4:])})
		// fmt.Println("egress_port: ", binary.BigEndian.Uint32(rep[0:4]))
		// fmt.Println("instanse   : ", binary.BigEndian.Uint32(rep[4:]))
	}

	for idx, r := range replicaTest {
		fmt.Println(idx, r)
		// fmt.Println("egress_port: ", r.EgressPort)
		// fmt.Println("instanse   : ", r.Instance)
	}
}
*/

/*
func main() {
	actions := make([]map[string]interface{}, 0)

	action := make(map[string]interface{}, 0)
	action["id"] = uint32(10)
	action["name"] = "action.sample"
	action["params"] = make([]map[string]interface{}, 2)

	var param map[string]interface{}

	param = make(map[string]interface{}, 0)
	param["id"] = uint32(10)
	param["value"] = []byte{uint8(4), uint8(4)}
	action["params"].([]map[string]interface{})[0] = param

	param = make(map[string]interface{}, 0)
	param["id"] = uint32(11)
	param["value"] = []byte{uint8(8), uint8(8)}
	action["params"].([]map[string]interface{})[1] = param

	actions = append(actions, action)

	fmt.Println(actions)
}
*/
