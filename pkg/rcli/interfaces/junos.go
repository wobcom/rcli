package interfaces

import (
	"context"
	"fmt"
	"github.com/damianoneill/net/v2/netconf/client"
	"github.com/damianoneill/net/v2/netconf/common"
	"github.com/gookit/slog"
	"github.com/wobcom/router-cli/pkg/rcli/api"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"net"
	"os"
)

type JunosInterface struct {
	RouterAddress string
	User          string
	RPCSession    client.Session
}

func NewJunosInterface(routerAddress string, user string) (*JunosInterface, error) {
	jI := JunosInterface{
		RouterAddress: routerAddress,
		User:          user,
		RPCSession:    nil,
	}

	err := jI.Connect()
	if err != nil {
		return nil, err
	}

	return &jI, nil
}

func (j *JunosInterface) DoRequest(req common.Request) (string, error) {
	r, err := j.RPCSession.Execute(req)
	if err != nil {
		return "", err
	}

	return r.Data, nil
}

func getSSHAgentAuthMethod() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func (j *JunosInterface) Connect() error {

	sshAgentAuthMethod := getSSHAgentAuthMethod()
	sshConfig := &ssh.ClientConfig{
		User: j.User,
		Auth: []ssh.AuthMethod{
			sshAgentAuthMethod,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	slog.Info(fmt.Sprintf("Creating NETCONF session with %v as %v", j.RouterAddress, j.User))

	s, err := client.NewRPCSession(context.Background(), sshConfig, j.RouterAddress)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Authenticated as %v", j.User))

	j.RPCSession = s

	return nil
}

func (j *JunosInterface) Close() {
	j.RPCSession.Close()
}

func (j *JunosInterface) GetVersion() (string, error) {
	resp, err := j.ExecuteCommand("show version", "json")

	if err != nil {
		return "", err
	}

	return api.ParseJunosVersionFromJson(resp)
}

func (j *JunosInterface) GetConfiguration() (*api.JunosConfiguration, error) {

	rpcReq := common.Request(`
	   	<get-configuration 
			format="text"
		>
		</get-configuration>
	`)

	res, err := j.DoRequest(rpcReq)
	if err != nil {
		return nil, err
	}

	junosConf, err := api.ParseFromText(res)
	return junosConf, err
}

func (j *JunosInterface) LoadConfiguration(junosConfiguration *api.JunosConfiguration) error {
	slog.Info("Loading configuration onto router as candidate configuration")
	rpcReq := common.Request(fmt.Sprintf(`
		<load-configuration
			action="override"
			format="text"
        >	    
			%v
		</load-configuration>
    `, junosConfiguration.ToText()))

	loadRes, err := j.DoRequest(rpcReq)
	if err != nil {
		return err
	}

	err = api.ParseLoadConfigurationResultsFromText(loadRes)
	if err != nil {
		return err
	}

	return err
}

func (j *JunosInterface) DiffConfiguration() (*api.JunosDiff, error) {
	slog.Info("Diffing candidate configuration against running configuration")

	rpcReq := common.Request(`
		<get-configuration compare="rollback" rollback="0" format="text"/>
	`)
	diffResp, err := j.DoRequest(rpcReq)
	if err != nil {
		return nil, err
	}

	junosDiff, err := api.ParseDiffFromText(diffResp)

	return junosDiff, err
}

func (j *JunosInterface) CommitConfiguration() error {
	slog.Info("Committing candidate configuration, needs to be confirmed afterwards")

	rpcReq := common.Request(`
	   	<commit-configuration>
			<confirmed/>
        	<confirm-timeout>5</confirm-timeout>
		</commit-configuration>
	`)

	_, err := j.DoRequest(rpcReq)
	return err
}

func (j *JunosInterface) ConfirmConfiguration() error {
	slog.Info("Confirming configuration")

	rpcReq := common.Request(`
	   	<commit-configuration/>
	`)

	_, err := j.DoRequest(rpcReq)
	return err
}

func (j *JunosInterface) ExecuteCommand(command string, format string) (string, error) {
	slog.Info(fmt.Sprintf("Executing %v", command))

	rpcReq := common.Request(fmt.Sprintf(`
	   	<command format="%v">
			%v
		</command>
	`, format, command))

	rpcRes, err := j.DoRequest(rpcReq)
	if err != nil {
		return "", err
	}
	commandResult, err := api.ParseCommandResultsFromText(rpcRes, format)
	return commandResult, err
}
