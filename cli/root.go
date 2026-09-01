package cli

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/dcc-bigfred/rb/app"
	"github.com/dcc-bigfred/rb/decoder"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	var debug bool

	command := &cobra.Command{
		Use:   "rb",
		Short: "CLI for RailBOX RB23xx decoder sound slots",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
		},
		RunE: func(command *cobra.Command, args []string) error {
			return errors.New("please select a command")
		},
	}

	command.PersistentFlags().BoolVarP(&debug, "debug", "v", false, "Increase verbosity to the debug level")
	command.AddCommand(NewSoundCommand())
	return command
}

func NewSoundCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "sound",
		Short: "Sound management for RailBOX RB23xx decoders",
		RunE: func(command *cobra.Command, args []string) error {
			return errors.New("please select a command")
		},
	}

	command.AddCommand(NewSoundClearCommand())
	command.AddCommand(NewSoundSyncCommand())
	return command
}

func NewSoundClearCommand() *cobra.Command {
	type Args struct {
		Timeout uint16
	}
	cmdArgs := Args{}

	command := &cobra.Command{
		Use:   "clear <slot>",
		Short: "Clear sound files from a slot on the RailBOX RB23xx decoder",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			slot64, err := strconv.ParseUint(args[0], 10, 8)
			if err != nil {
				return fmt.Errorf("invalid slot number %q: %w", args[0], err)
			}

			return app.ClearSoundSlot(uint8(slot64), decoder.WithTimeout(cmdArgs.Timeout))
		},
	}

	command.Flags().Uint16VarP(&cmdArgs.Timeout, "timeout", "", 10, "HTTP connection timeout in seconds")
	return command
}

func NewSoundSyncCommand() *cobra.Command {
	type Args struct {
		Timeout     uint16
		DryRun      bool
		WithoutLast bool
		Watch       bool
	}
	cmdArgs := Args{}

	command := &cobra.Command{
		Use:   "sync <slot> <local-dir>",
		Short: "Synchronise a local directory with a sound slot on the RailBOX RB23xx decoder",
		Long: `Compares the contents of a local directory with the given sound slot on the decoder.
Files present locally but missing on the decoder are uploaded.
Files present on the decoder but missing locally are deleted from the decoder.
Files present on both sides but differing in size are re-uploaded.
By default the 5 most recently modified local files (modified within the last 24 h) are always re-uploaded.
Use --without-last to disable this behaviour.
Use --watch to keep watching the directory and re-sync automatically on every change.`,
		Args: cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			slot64, err := strconv.ParseUint(args[0], 10, 8)
			if err != nil {
				return fmt.Errorf("invalid slot number %q: %w", args[0], err)
			}

			opts := []decoder.Option{decoder.WithTimeout(cmdArgs.Timeout)}

			if cmdArgs.Watch {
				return app.WatchSoundSlot(uint8(slot64), args[1], cmdArgs.DryRun, cmdArgs.WithoutLast, opts...)
			}
			return app.SyncSoundSlot(uint8(slot64), args[1], cmdArgs.DryRun, cmdArgs.WithoutLast, opts...)
		},
	}

	command.Flags().Uint16VarP(&cmdArgs.Timeout, "timeout", "", 10, "HTTP connection timeout in seconds")
	command.Flags().BoolVar(&cmdArgs.DryRun, "dry-run", false, "Preview changes without uploading or deleting any files")
	command.Flags().BoolVarP(&cmdArgs.WithoutLast, "without-last", "l", false, "Disable automatic re-upload of the 5 most recently modified files (last 24 h)")
	command.Flags().BoolVarP(&cmdArgs.Watch, "watch", "w", false, "Watch the local directory and re-sync automatically on every file change")

	return command
}
