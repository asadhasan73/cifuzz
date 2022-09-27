package init

import (
	_ "embed"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/log"
)

//go:embed instructions/cmake
var cmakeSetup string

//go:embed instructions/maven
var mavenSetup string

//go:embed instructions/gradle
var gradleSetup string

func New() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Set up a project for use with cifuzz",
		Long: `This command sets up a project for use with cifuzz, creating a
'cifuzz.yaml' config file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}

	cmdutils.DisableConfigCheck(initCmd)

	return initCmd
}

func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}
	log.Debugf("Using current working directory: %s", cwd)

	configpath, err := config.CreateProjectConfig(cwd)
	if err != nil {
		// explicitly inform the user about an existing config file
		if errors.Is(err, os.ErrExist) && configpath != "" {
			log.Warnf("Config already exists in %s", configpath)
			err = cmdutils.ErrSilent
		}
		log.Error(err, "Failed to create config")
		return err
	}
	log.Successf("Configuration saved in %s", configpath)

	setUpAndMentionBuildSystemIntegrations(cwd)

	log.Print(`
Use 'cifuzz create' to create your first fuzz test.`)
	return nil
}

func setUpAndMentionBuildSystemIntegrations(cwd string) {
	// Printing build system instructions is best-effort: Do not fail on errors.
	buildSystem, err := config.DetermineBuildSystem(cwd)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Debug(err)
		}
		return
	}

	switch buildSystem {
	case config.BuildSystemCMake:
		// Note: We set NO_SYSTEM_ENVIRONMENT_PATH to avoid that the
		// system-wide cmake package takes precedence over a package
		// from a per-user installation (which is what we want, per-user
		// installations should usually take precedence over system-wide
		// installations).
		//
		// The find_package search procedure is described in
		// https://cmake.org/cmake/help/latest/command/find_package.html#config-mode-search-procedure.
		//
		// Without NO_SYSTEM_ENVIRONMENT_PATH, find_package looks in
		// paths with prefixes from the PATH environment variable in
		// step 5 (omitting any trailing "/bin").
		// The PATH usually includes "/usr/local/bin", which means that
		// find_package searches in "/usr/local/share/cifuzz" in this
		// step, which is the path we use for a system-wide installation.
		//
		// The per-user directory is searched in step 6.
		//
		// With NO_SYSTEM_ENVIRONMENT_PATH, the system-wide installation
		// directory is only searched in step 7.
		log.Print(cmakeSetup)
	case config.BuildSystemMaven:
		log.Print(mavenSetup)
	case config.BuildSystemGradle:
		log.Print(gradleSetup)
	}
}
