package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/taka-sho/teraflow/internal/state"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current project stage and phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := configPathFromCmd(cmd)
			if err != nil {
				return err
			}
			format, err := outputFormatFromCmd(cmd)
			if err != nil {
				return err
			}

			s, err := state.LoadState(configPath)
			if err != nil {
				var pe *os.PathError
				if errors.As(err, &pe) && errors.Is(pe.Err, os.ErrNotExist) {
					fmt.Fprintln(cmd.ErrOrStderr(), notProjectError)
					return errors.New(notProjectError)
				}
				return err
			}

			if format == "json" {
				return writeJSON(cmd, map[string]string{
					"stage": s.Lifecycle.CurrentStage,
					"phase": s.Phases.Current,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Stage:  %s\n", s.Lifecycle.CurrentStage)
			fmt.Fprintf(cmd.OutOrStdout(), "Phase:  %s\n", s.Phases.Current)
			return nil
		},
	}
}
