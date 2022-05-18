// Copyright 2018 The xfsgo Authors
// This file is part of the xfsgo library.
//
// The xfsgo library is free software: you can redistribute it and/or modify
// it under the terms of the MIT Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The xfsgo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// MIT Lesser General Public License for more details.
//
// You should have received a copy of the MIT Lesser General Public License
// along with the xfsgo library. If not, see <https://mit-license.org/>.

package sub

import (
	"fmt"
	"os"
	"xfsgo"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	rpchost string
	rootCmd = &cobra.Command{
		Use:                   fmt.Sprintf("%s <command> [<options>]", xfsgo.GetAppName()),
		DisableFlagsInUseLine: true,
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	versionCmd = &cobra.Command{
		Use:                   "version",
		Short:                 fmt.Sprintf("Print the version number of %s", xfsgo.GetAppName()),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", err))
		os.Exit(1)
	}
}

func showVersion() {
	fmt.Println(xfsgo.VersionString())
}

func helpTmpl() string {
	return `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

func usageTmpl() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}

Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] -h,--help" for more information about a command.{{end}}
`
}

func initCliSubCommands() {
	chainflags := chainCommand.PersistentFlags()
	chainflags.StringVarP(&rpchost, "host", "", "", "Set rpc api host")
	minerflags := minerCommand.PersistentFlags()
	minerflags.StringVarP(&rpchost, "host", "", "", "Set rpc api host")
	netflasg := netCommand.PersistentFlags()
	netflasg.StringVarP(&rpchost, "host", "", "", "Set rpc api host")
	stateflags := getStateCommand.PersistentFlags()
	stateflags.StringVarP(&rpchost, "host", "", "", "Set rpc api host")
	txpoolflags := getTxpoolCommand.PersistentFlags()
	txpoolflags.StringVarP(&rpchost, "host", "", "", "Set rpc api host")
	walletflags := walletCommand.PersistentFlags()
	walletflags.StringVarP(&rpchost, "host", "", "", "Set rpc api host")

}

func init() {
	mFlags := rootCmd.PersistentFlags()
	mFlags.StringVarP(&cfgFile, "config", "C", "", "Set config file")
	rootCmd.SetHelpTemplate(helpTmpl())
	rootCmd.SetUsageTemplate(usageTmpl())
	rootCmd.AddCommand(versionCmd)
	initCliSubCommands()
}
