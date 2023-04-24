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

func cliDiffPrev(jI *interfaces.JunosInterface, localFile string, loadAction string, diffFile *string) (*api.JunosDiff, error) {

	version, err := jI.GetVersion()
	if err != nil {
		return nil, err
	}
	newConf, err := api.ParseFromFile(localFile, version)
	if err != nil {
		return nil, err
	}

	err = jI.LoadConfiguration(newConf, loadAction)
	if err != nil {
		return nil, err
	}

	diff, err := jI.DiffConfiguration()
	if err != nil {
		return nil, err
	}

	if diffFile != nil {
		err := diff.WriteToFile(*diffFile)
		if err != nil {
			return nil, err
		}
	}

	diff.Print()

	return diff, nil
}

type ApplyCommand struct {
	Yes        bool   `help:"Skips interactive diff reviewing, meant to use within CI environments"`
	Commit     bool   `help:"Skips wait for commit and commits configuration after upload"`
	LoadAction string `enum:"override,replace" default:"override"`

	Router    string `arg:"" name:"router"`
	LocalFile string `arg:"" name:"local_file" type:"existing_file"`
}

func (aC *ApplyCommand) Run(cc CommandContext) error {
	jI, err := interfaces.NewJunosInterface(fmt.Sprintf("%v:830", aC.Router), cc.User)
	if err != nil {
		return err
	}
	defer jI.Close()

	err = jI.LockingConfiguration(func() error {

		diff, err := cliDiffPrev(jI, aC.LocalFile, aC.LoadAction, nil)
		if err != nil {
			return err
		}

		if diff != nil {
			slog.Warn("No changes found")
			return nil
		}

		if !aC.Yes {
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
		} else {
			slog.Debug("--yes was used, skipping interactive confirmation dialog...")
		}

		err = jI.CommitConfiguration()
		if err != nil {
			return err
		}

		if !aC.Commit {
			slog.Info("Waiting 3 minutes before confirming configuration...")
			time.Sleep(3 * time.Minute)
		} else {
			slog.Debug("--commit was used, skipping waiting period...")
		}
		err = jI.ConfirmConfiguration()
		return err
	})

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
	DiffOutput *string `help:"Writes the diff into a specified file" short:"f"`
	LoadAction string  `enum:"override,replace" default:"override"`

	Router    string `arg:"" name:"router"`
	LocalFile string `arg:"" name:"local_file"`
}

func (c *CheckCommand) Run(cc CommandContext) error {
	jI, err := interfaces.NewJunosInterface(fmt.Sprintf("%v:830", c.Router), cc.User)
	if err != nil {
		return err
	}
	defer jI.Close()

	err = jI.LockingConfiguration(func() error {
		_, err := cliDiffPrev(jI, c.LocalFile, c.LoadAction, c.DiffOutput)
		return err
	})
	if err != nil {
		return err
	}

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
		logger.Output = os.Stderr

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
