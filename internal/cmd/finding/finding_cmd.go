package finding

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/cmdutils"
	"code-intelligence.com/cifuzz/pkg/finding"
	"code-intelligence.com/cifuzz/pkg/log"
)

type cmdOpts struct {
}

func New() *cobra.Command {
	opts := &cmdOpts{}
	findingCmd := &cobra.Command{
		Use:     "finding",
		Aliases: []string{"findings"},
		Short:   "List and show findings",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, opts)
		},
	}

	return findingCmd
}

func run(cmd *cobra.Command, args []string, opts *cmdOpts) (err error) {
	projectDir, err := config.FindProjectDir()
	if errors.Is(err, os.ErrNotExist) {
		// The project directory doesn't exist, this is an expected
		// error, so we print it and return a silent error to avoid
		// printing a stack trace
		log.Error(err, fmt.Sprintf("%s\nUse 'cifuzz init' to set up a project for use with cifuzz.", err.Error()))
		return cmdutils.ErrSilent
	}
	if err != nil {
		return err
	}

	if len(args) == 0 {
		findings, err := finding.ListFindings(projectDir)
		if err != nil {
			return err
		}
		if len(findings) == 0 {
			log.Print("This project doesn't have any findings yet")
			return nil
		}
		for _, f := range findings {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), f.Name)
		}
	}

	return nil
}
