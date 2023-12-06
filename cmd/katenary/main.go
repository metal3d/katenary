package main

import (
	"fmt"
	"katenary/generator"
	"katenary/utils"
	"os"
	"strings"

	"github.com/compose-spec/compose-go/cli"
	"github.com/spf13/cobra"
)

const longHelp = `Katenary is a tool to convert compose files to Helm Charts.

Each [command] and subcommand has got an "help" and "--help" flag to show more information.
`

func main() {
	// The base command
	rootCmd := &cobra.Command{
		Use:   "katenary",
		Long:  longHelp,
		Short: "Katenary is a tool to convert docker-compose files to Helm Charts",
	}
	rootCmd.Example = `  katenary convert -c docker-compose.yml -o ./charts`

	rootCmd.Version = generator.Version
	rootCmd.CompletionOptions.DisableDescriptions = false
	rootCmd.CompletionOptions.DisableNoDescFlag = false

	rootCmd.AddCommand(
		generateCompletionCommand(rootCmd.Name()),
		generateVersionCommand(),
		generateConvertCommand(),
		generateHashComposefilesCommand(),
		generateLabelHelpCommand(),
	)

	rootCmd.Execute()
}

const completionHelp = `To load completions:

Bash:
  # Add this line in your ~/.bashrc or ~/.bash_profile file
  $ source <(%[1]s completion bash)

  # Or, you can load completions for each users session. Execute once:
  # Linux:
  $ %[1]s completion bash > /etc/bash_completion.d/%[1]s
  # macOS:
  $ %[1]s completion bash > $(brew --prefix)/etc/bash_completion.d/%[1]s

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ %[1]s completion zsh > "${fpath[1]}/_%[1]s"

  # You will need to start a new shell for this setup to take effect.

fish:
  $ %[1]s completion fish | source

  # To load completions for each session, execute once:
  $ %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

PowerShell:
  PS> %[1]s completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> %[1]s completion powershell > %[1]s.ps1
  # and source this file from your PowerShell profile.
`

func generateCompletionCommand(name string) *cobra.Command {
	bashV1 := false
	cmd := &cobra.Command{
		Use:                   "completion",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short:                 "Generates completion scripts",
		Long:                  fmt.Sprintf(completionHelp, name),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}
			switch args[0] {
			case "bash":
				// get the bash version
				if cmd.Flags().Changed("bash-v1") {
					cmd.Root().GenBashCompletion(os.Stdout)
					return
				}
				cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
		},
	}

	// add a flag to force bash completion v1
	cmd.Flags().Bool("bash-v1", bashV1, "Force bash completion v1")

	return cmd
}

func generateConvertCommand() *cobra.Command {
	force := false
	outputDir := "./chart"
	dockerComposeFile := make([]string, 0)
	profiles := make([]string, 0)
	helmdepUpdate := false
	var appVersion *string
	givenAppVersion := ""
	chartVersion := "0.1.0"

	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Converts a docker-compose file to a Helm Chart",
		Run: func(cmd *cobra.Command, args []string) {
			if givenAppVersion != "" {
				appVersion = &givenAppVersion
			}
			generator.Convert(generator.ConvertOptions{
				Force:        force,
				OutputDir:    outputDir,
				Profiles:     profiles,
				HelmUpdate:   helmdepUpdate,
				AppVersion:   appVersion,
				ChartVersion: chartVersion,
			}, dockerComposeFile...)
		},
	}

	convertCmd.Flags().BoolVarP(&force, "force", "f", force, "Force the overwrite of the chart directory")
	convertCmd.Flags().BoolVarP(&helmdepUpdate, "helm-update", "u", helmdepUpdate, "Update helm dependencies if helm is installed")
	convertCmd.Flags().StringSliceVarP(&profiles, "profile", "p", profiles, "Specify the profiles to use")
	convertCmd.Flags().StringVarP(&outputDir, "output-dir", "o", outputDir, "Specify the output directory")
	convertCmd.Flags().StringSliceVarP(&dockerComposeFile, "compose-file", "c", cli.DefaultFileNames, "Specify an alternate compose files - can be specified multiple times or use coma to separate them")
	convertCmd.Flags().StringVarP(&givenAppVersion, "app-version", "a", "", "Specify the app version (in Chart.yaml)")
	convertCmd.Flags().StringVarP(&chartVersion, "chart-version", "v", chartVersion, "Specify the chart version (in Chart.yaml)")
	return convertCmd

}

func generateVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Katenary",
		Run: func(cmd *cobra.Command, args []string) {
			println(generator.Version)
		},
	}
}

func generateLabelHelpCommand() *cobra.Command {

	markdown := false
	all := false
	cmd := &cobra.Command{
		Use:   "help-labels [label]",
		Short: "Print the labels help for all or a specific label",
		Long: `Print the labels help for all or a specific label
If no label is specified, the help for all labels is printed.
If a label is specified, the help for this label is printed.

The name of the label must be specified without the prefix ` + generator.KATENARY_PREFIX + `.

e.g. 
  kanetary help-labels
  katenary help-labels ingress
  katenary help-labels map-env
`,
		ValidArgs: generator.GetLabelNames(),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				fmt.Println(generator.GetLabelHelpFor(args[0], markdown))
				return
			}
			if all {
				// show the help for all labels
				l := len(generator.GetLabelNames())
				for i, label := range generator.GetLabelNames() {
					fmt.Println(generator.GetLabelHelpFor(label, markdown))
					if !markdown && i < l-1 {
						fmt.Println(strings.Repeat("-", 80))
					}
				}
				return
			}
			fmt.Println(generator.GetLabelHelp(markdown))
		},
	}

	cmd.Flags().BoolVarP(&markdown, "markdown", "m", markdown, "Use the markdown format")
	cmd.Flags().BoolVarP(&all, "all", "a", all, "Print the full help for all labels")

	return cmd
}

func generateHashComposefilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hash-composefiles [composefile]",
		Short: "Print the hash of the composefiles",
		Long: `Print the hash of the composefiles
If no composefile is specified, the hash of all composefiles is printed.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				if hash, err := utils.HashComposefiles(args); err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(hash)
				}
				return
			}
		},
	}
	return cmd
}
