// Package cli provides CLI utilities for bifrost-extensions using phenotype-go-kit.
package cli

import (
	"github.com/KooshaPari/phenotype-go-kit/pkg/cli"
	"github.com/spf13/cobra"
)

// CreateRootCommand creates the root command for bifrost-extensions using phenotype-go-kit's CLI utilities.
//
// Parameters:
//   - version: The version of bifrost-extensions
//
// Returns:
//   - *cobra.Command: The root command
func CreateRootCommand(version string) *cobra.Command {
	return cli.CreateRootCommand(
		cli.RootCommandConfig{
			Name:     "bifrost-extensions",
			Short:    "Bifrost Extensions CLI",
			Long:     `Bifrost Extensions - Extended capabilities and provider integrations for Bifrost`,
			Version:  version,
			Examples: "",
		},
		func(cmd *cobra.Command, args []string) error {
			// Default behavior: show help if no subcommand is provided
			return cmd.Help()
		},
	)
}

// CommandBuilder provides a fluent interface for building bifrost-extensions commands.
type CommandBuilder struct {
	builder *cli.CommandBuilder
}

// NewCommandBuilder creates a new CommandBuilder.
//
// Parameters:
//   - use: The command name
//
// Returns:
//   - *CommandBuilder: A new CommandBuilder instance
func NewCommandBuilder(use string) *CommandBuilder {
	return &CommandBuilder{
		builder: cli.NewCommandBuilder(use),
	}
}

// Short sets the short description of the command.
func (cb *CommandBuilder) Short(short string) *CommandBuilder {
	cb.builder.Short(short)
	return cb
}

// Long sets the long description of the command.
func (cb *CommandBuilder) Long(long string) *CommandBuilder {
	cb.builder.Long(long)
	return cb
}

// Examples sets the usage examples of the command.
func (cb *CommandBuilder) Examples(examples string) *CommandBuilder {
	cb.builder.Examples(examples)
	return cb
}

// RunE sets the RunE function of the command.
func (cb *CommandBuilder) RunE(runFunc func(cmd *cobra.Command, args []string) error) *CommandBuilder {
	cb.builder.RunE(runFunc)
	return cb
}

// Build returns the constructed cobra command.
func (cb *CommandBuilder) Build() *cobra.Command {
	return cb.builder.Build()
}

// AddStringFlag adds a string flag to the command.
func (cb *CommandBuilder) AddStringFlag(name string, shorthand string, defaultValue string, usage string) *CommandBuilder {
	cb.builder.StringFlag(name, shorthand, defaultValue, usage)
	return cb
}

// AddBoolFlag adds a boolean flag to the command.
func (cb *CommandBuilder) AddBoolFlag(name string, shorthand string, defaultValue bool, usage string) *CommandBuilder {
	cb.builder.BoolFlag(name, shorthand, defaultValue, usage)
	return cb
}

// AddIntFlag adds an integer flag to the command.
func (cb *CommandBuilder) AddIntFlag(name string, shorthand string, defaultValue int, usage string) *CommandBuilder {
	cb.builder.IntFlag(name, shorthand, defaultValue, usage)
	return cb
}

// AddSubcommand adds a subcommand to the command.
func (cb *CommandBuilder) AddSubcommand(subCmd *cobra.Command) *CommandBuilder {
	cb.builder.AddSubcommand(subCmd)
	return cb
}
