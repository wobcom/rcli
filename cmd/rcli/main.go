package main

import (
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/gookit/slog"
	"github.com/manifoldco/promptui"
	"github.com/wobcom/router-cli/pkg/rcli/api"
	"github.com/wobcom/router-cli/pkg/rcli/interfaces"
	"os"
	"strings"
	"time"
)

type CommandContext struct {
	User string
}

type ApplyCommand struct {
	Router    string `arg:"" name:"router"`
	LocalFile string `arg:"" name:"local_file" type:"existing_file"`
}

func cliDiffPrev(jI *interfaces.JunosInterface, localFile string) error {

	version, err := jI.GetVersion()
	if err != nil {
		return err
	}
	newConf, err := api.ParseFromFile(localFile, version)
	if err != nil {
		return err
	}

	err = jI.LoadConfiguration(newConf)
	if err != nil {
		return err
	}

	diff, err := jI.DiffConfiguration()
	if err != nil {
		return err
	}

	diff.Print()
	return nil
}

func (aC *ApplyCommand) Run(cc CommandContext) error {
	jI, err := interfaces.NewJunosInterface(fmt.Sprintf("%v:830", aC.Router), cc.User)
	if err != nil {
		return err
	}
	defer jI.Close()

	err = cliDiffPrev(jI, aC.LocalFile)
	if err != nil {
		return nil
	}

	doApplyPrompt := promptui.Select{
		Label: fmt.Sprintf("Do you want to apply this configuration onto %v?", aC.Router),
		Items: []string{
			"No",
			"Yes",
		},
		HideHelp: true,
	}

	_, result, err := doApplyPrompt.Run()

	if err != nil || result != "Yes" {
		return nil
	}

	err = jI.CommitConfiguration()
	if err != nil {
		return err
	}

	slog.Info("Waiting 3 minutes before confirming configuration...")
	time.Sleep(3 * time.Minute)
	err = jI.ConfirmConfiguration()
	if err != nil {
		return err
	}

	return nil
}

type ExecCommand struct {
	Output string `enum:"text,xml,json" default:"text" short:"o"`

	Router string `arg:"" name:"router"`

	Command []string `arg:""`
}

func (eC *ExecCommand) Run(cc CommandContext) error {

	jI, err := interfaces.NewJunosInterface(fmt.Sprintf("%v:830", eC.Router), cc.User)
	if err != nil {
		return err
	}
	defer jI.Close()

	result, err := jI.ExecuteCommand(strings.Join(eC.Command, " "), eC.Output)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

type CheckCommand struct {
	Router    string `arg:"" name:"router"`
	LocalFile string `arg:"" name:"local_file"`
}

func (c *CheckCommand) Run(cc CommandContext) error {
	jI, err := interfaces.NewJunosInterface(fmt.Sprintf("%v:830", c.Router), cc.User)
	if err != nil {
		return err
	}
	defer jI.Close()

	err = cliDiffPrev(jI, c.LocalFile)
	return err
}

type CommandLineInterface struct {
	User string `help:"NETCONF ssh user" env:"USER" short:"u"`

	Check CheckCommand `cmd:"" help:"Loads local configuration onto router and shows a diff."`
	Apply ApplyCommand `cmd:"" help:"Applies local configuration file on router"`
	Exec  ExecCommand  `cmd:"" help:"Executes an given command on the router"`
}

func innerMain() error {

	slog.Configure(func(logger *slog.SugaredLogger) {
		f := logger.Formatter.(*slog.TextFormatter)

		myTemplate := "[{{datetime}}] [{{level}}] {{message}}\n"
		f.SetTemplate(myTemplate)
		f.EnableColor = true
	})

	var cli CommandLineInterface
	ctx := kong.Parse(&cli)

	cCtx := CommandContext{
		User: cli.User,
	}

	// Call the Run() method of the selected parsed command.
	err := ctx.Run(cCtx)
	slog.ErrorT(err)

	return err
}

func main() {
	err := innerMain()
	if err != nil {
		os.Exit(1)
	}
}
