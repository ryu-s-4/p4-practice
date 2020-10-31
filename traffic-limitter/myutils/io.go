package myutils

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
)

// ControllerInfo is information for the controller
type ControllerInfo struct {
	deviceid    uint64
	roleid      uint64
	electionid  v1.Uint128
	p4infoPath  string
	devconfPath string
	runconfPath string
}

// MasterArbitrationUpdate gets arbitration for the master
func MasterArbitrationUpdate(cntlInfo ControllerInfo, ch v1.P4Runtime_StreamChannelClient) (*v1.MasterArbitrationUpdate, error) {

	request := v1.StreamMessageRequest{
		Update: &v1.StreamMessageRequest_Arbitration{
			Arbitration: &v1.MasterArbitrationUpdate{
				DeviceId:   cntlInfo.deviceid,
				ElectionId: &cntlInfo.electionid,
			},
		},
	}

	err := ch.Send(&request)
	if err != nil {
		return nil, err
	}

	response, err := ch.Recv()
	if err != nil {
		return nil, err
	}

	updateResponse := response.GetUpdate()
	switch updateResponse.(type) {

	case *v1.StreamMessageResponse_Arbitration:
		arbitration := response.GetArbitration()
		return arbitration, nil

	case *v1.StreamMessageResponse_Packet:
		/* TODO */
		/* packet := response.GetPacket() */

	case *v1.StreamMessageResponse_Digest:
		/* TODO */
		/* digest := response.GetDigest() */

	case *v1.StreamMessageResponse_IdleTimeoutNotification:
		/* TODO */
		/* idletimenotf := response.GetIdleTimeoutNotification() */
	}

	/* unknown update response type is received. */
	return nil, fmt.Errorf("unknown update response type")
}

// MyCreateConfig creates config data for SetForwardingPipelineConfig
func MyCreateConfig(p4infoPath string, devconfPath string) (*v1.ForwardingPipelineConfig, error) {

	p4info := config_v1.P4Info{}
	p4infoBytes, err := ioutil.ReadFile(p4infoPath)
	if err != nil {
		return nil, err
	}
	proto.UnmarshalText(string(p4infoBytes), &p4info)

	var devconf []byte
	devconf, err = ioutil.ReadFile(devconfPath)
	if err != nil {
		return nil, err
	}

	forwardingpipelineconfig := v1.ForwardingPipelineConfig{
		P4Info:         &p4info,
		P4DeviceConfig: devconf}

	return &forwardingpipelineconfig, nil
}

// SetForwardingPipelineConfig sets the user defined configuration to the data plane.
func SetForwardingPipelineConfig(
	cntlInfo ControllerInfo,
	actionType string,
	client v1.P4RuntimeClient) (*v1.SetForwardingPipelineConfigResponse, error) {

	var action v1.SetForwardingPipelineConfigRequest_Action
	switch actionType {
	case "VERIFY":
		action = v1.SetForwardingPipelineConfigRequest_VERIFY
	case "VERIFY_AND_SAVE":
		action = v1.SetForwardingPipelineConfigRequest_VERIFY_AND_SAVE
	case "VERIFY_AND_COMMIT":
		action = v1.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT
	case "COMMIT":
		action = v1.SetForwardingPipelineConfigRequest_COMMIT
	case "RECONCILE_AND_COMMIT":
		action = v1.SetForwardingPipelineConfigRequest_RECONCILE_AND_COMMIT
	default:
		action = v1.SetForwardingPipelineConfigRequest_UNSPECIFIED
	}

	config, err := MyCreateConfig(cntlInfo.p4infoPath, cntlInfo.devconfPath)
	if err != nil {
		return nil, err
	}

	request := v1.SetForwardingPipelineConfigRequest{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid,
		Action:     action,
		Config:     config}

	response, err := client.SetForwardingPipelineConfig(context.TODO(), &request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// SendWriteRequest sends write request to the data plane.
func SendWriteRequest(
	cntlInfo ControllerInfo,
	updates []*v1.Update,
	atomisityType string,
	client v1.P4RuntimeClient) (*v1.WriteResponse, error) {

	var atomisity v1.WriteRequest_Atomicity
	switch atomisityType {
	case "CONTINUE_ON_ERROR":
		atomisity = v1.WriteRequest_CONTINUE_ON_ERROR
	case "ROLLBACK_ON_ERROR": // OPTIONAL
		atomisity = v1.WriteRequest_ROLLBACK_ON_ERROR
	case "DATAPLANE_ATOMIC": // OPTIONAL
		atomisity = v1.WriteRequest_DATAPLANE_ATOMIC
	default:
		atomisity = v1.WriteRequest_CONTINUE_ON_ERROR
	}

	writeRequest := v1.WriteRequest{
		DeviceId:   cntlInfo.deviceid,
		ElectionId: &cntlInfo.electionid,
		Updates:    updates,
		Atomicity:  atomisity,
	}

	writeResponse, err := client.Write(context.TODO(), &writeRequest)
	if err != nil {
		return nil, err
	}

	return writeResponse, nil
}

// CreateReadClient creates New ReadClient.
func CreateReadClient(
	cntlInfo ControllerInfo,
	entities []*v1.Entity,
	client v1.P4RuntimeClient) (v1.P4Runtime_ReadClient, error) {

	readRequest := v1.ReadRequest{
		DeviceId: cntlInfo.deviceid,
		Entities: entities,
	}

	readclient, err := client.Read(context.TODO(), &readRequest)
	if err != nil {
		return nil, err
	}

	return readclient, nil
}
