package myutils

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
)

// ControlPlaneClient ...
type ControlPlaneClient struct {
	deviceid   uint64
	roleid     uint64
	electionid v1.Uint128
	p4info     *config_v1.P4Info
	config     *v1.ForwardingPipelineConfig
	entries    *EntryHelper
	client     v1.P4RuntimeClient
	channel    v1.P4Runtime_StreamChannelClient
}

// InitConfig initializes P4Info / ForwardingPipelineConfig / EntryHelper for the ControlPlaneClient.
func (cp *ControlPlaneClient) InitConfig(p4infoPath string, devconfPath string, runconfPath string) error {

	// P4Info
	p4infoBytes, err := ioutil.ReadFile(p4infoPath)
	if err != nil {
		return err
	}
	err := proto.UnmarshalText(string(p4infoBytes), cp.p4info)
	if err != nil {
		return err
	}

	// ForwardingPipelineConfig
	devconf, err := ioutil.ReadFile(devconfPath)
	if err != nil {
		return err
	}
	cp.config = v1.ForwardingPipelineConfig{
		P4Info:         cp.p4info,
		P4DeviceConfig: devconf,
	}

	// EntryHelper
	runtime, err := ioutil.ReadFile(runconfPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(runtime, cp.entries); err != nil {
		return err
	}

	return nil
}

// InitChannel ...
func (cp *ControlPlaneClient) InitChannel() error {

	if cp.client != nil {
		ch, err := cp.client.StreamChannel(context.TODO())
		if err != nil {
			return err
		}
		cp.channel = ch
		return nil
	} else {
		return fmt.Errorf("P4RuntimeClient is NOT created")
	}
}

// MasterArbitrationUpdate gets arbitration for the master
func (cp *ControlPlaneClient) MasterArbitrationUpdate() (*v1.MasterArbitrationUpdate, error) {

	request := v1.StreamMessageRequest{
		Update: &v1.StreamMessageRequest_Arbitration{
			Arbitration: &v1.MasterArbitrationUpdate{
				DeviceId:   cp.deviceid,
				ElectionId: cp.electionid,
			},
		},
	}

	err := cp.channel.Send(&request)
	if err != nil {
		return nil, err
	}

	response, err := cp.channel.Recv()
	if err != nil {
		return nil, err
	}

	updateResponse := response.GetUpdate()
	switch updateResponse.(type) {

	case *v1.StreamMessageResponse_Arbitration:
		arbitration := response.GetArbitration()
		return arbitration, nil
	}

	/* unknown update response type is received. */
	return nil, fmt.Errorf("unknown update response type")
}

// SetForwardingPipelineConfig sets the user defined configuration to the data plane.
func (cp *ControlPlaneClient) SetForwardingPipelineConfig(actionType string) (*v1.SetForwardingPipelineConfigResponse, error) {

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

	request := v1.SetForwardingPipelineConfigRequest{
		DeviceId:   cp.deviceid,
		ElectionId: cp.electionid,
		Action:     action,
		Config:     cp.config}

	response, err := cp.client.SetForwardingPipelineConfig(context.TODO(), &request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// SendWriteRequest sends write request to the data plane.
func (cp *ControlPlaneClient) SendWriteRequest(updates []*v1.Update, atomisityType string) (*v1.WriteResponse, error) {

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

	request := v1.WriteRequest{
		DeviceId:   cp.deviceid,
		ElectionId: cp.electionid,
		Updates:    updates,
		Atomicity:  atomisity,
	}

	response, err := cp.client.Write(context.TODO(), &request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// CreateReadClient creates New ReadClient.
func (cp *ControlPlaneClient) CreateReadClient(entities []*v1.Entity) (v1.P4Runtime_ReadClient, error) {

	request := v1.ReadRequest{
		DeviceId: cp.deviceid,
		Entities: entities,
	}

	readclient, err := cp.client.Read(context.TODO(), &readRequest)
	if err != nil {
		return nil, err
	}
	return readclient, nil
}
