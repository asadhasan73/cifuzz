package e2e

import (
	"testing"

	"code-intelligence.com/cifuzz/e2e-tests"
)

var containerRemoteRunTests = &[]e2e.TestCase{
	{
		Description: "container remote-run command is not available without CIFUZZ_PRERELEASE flag",
		Command:     "container remote-run",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Failed().ErrorContains("unknown command \"container\" for \"cifuzz\"")
		},
	},
	{
		Description: "container remote-run command is available in --help output with CIFUZZ_PRERELEASE flag",
		Command:     "container remote-run",
		Args:        []string{"--help"},
		Environment: []string{"CIFUZZ_PRERELEASE=true"},
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().OutputContains("container")
		},
	},
	{
		Description:   "container remote-run command in a maven/gradle example folder is available with CIFUZZ_PRERELEASE flag and pushes it to a registry",
		Command:       "container remote-run",
		Args:          []string{"--registry localhost:5000/test/cifuzz com.example.FuzzTestCase::myFuzzTest -v"},
		Environment:   []string{"CIFUZZ_PRERELEASE=true"},
		SampleFolder:  []string{"examples/maven", "examples/gradle"},
		ToolsRequired: []string{"docker", "java", "maven"},
		SkipOnOS:      "windows",
		Assert: func(t *testing.T, output e2e.CommandOutput) {
			output.Success().ErrorContains("Created fuzz container image with ID ")
			output.Success().ErrorContains("Start uploading image ")
			output.Success().ErrorContains("to localhost:5000/test/cifuzz")
			output.Success().ErrorContains("The push refers to repository [localhost:5000/test/cifuzz]")
		},
	},
}

func TestContainerRemoteRun(t *testing.T) {
	e2e.RunTests(t, *containerRemoteRunTests)
}
