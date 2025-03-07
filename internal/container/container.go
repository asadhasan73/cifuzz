package container

import (
	"context"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"code-intelligence.com/cifuzz/pkg/log"
)

func Create(fuzzTest string) (string, error) {
	cli, err := getDockerClient()
	if err != nil {
		return "", err
	}

	hostConfig := &container.HostConfig{}
	containerConfig := &container.Config{
		Image:        "cifuzz",
		Cmd:          []string{fuzzTest},
		AttachStdout: true,
		AttachStderr: true,
	}

	if viper.GetBool("verbose") {
		containerConfig.Cmd = append(containerConfig.Cmd, "-v")
	}

	// Make the container sleep forever if the environment variable is set.
	// This is useful for debugging, as it allows to exec into the container,
	// run the command manually and debug things in the container.
	if os.Getenv("CIFUZZ_CONTAINER_SLEEP") != "" {
		containerConfig.Env = append(containerConfig.Env, "CMD="+strings.Join(containerConfig.Cmd, " "))
		containerConfig.Entrypoint = []string{"sleep", "infinity"}
		containerConfig.Cmd = []string{}
		// When overwriting the command via Config.Cmd or Config.Entrypoint,
		// docker executes the command via a shell, which has the effect that
		// signals are not forwarded to the child process. To work around
		// this, we set Init to true, which causes docker to execute an init
		// process which forwards signals to the child process.
		useInit := true
		hostConfig.Init = &useInit
	}

	ctx := context.Background()
	cont, err := cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		&v1.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
		"", // TODO: should the container have a name?
	)
	if err != nil {
		return "", errors.WithStack(err)
	}

	log.Debugf("Created fuzz container %s based on image %s", cont.ID, containerConfig.Image)
	return cont.ID, nil
}

func Run(id string, outW, errW io.Writer) error {
	ctx := context.Background()

	cli, err := getDockerClient()
	if err != nil {
		return err
	}

	sigc := make(chan os.Signal, 128)
	signal.Notify(sigc)
	go forwardAllSignals(ctx, cli, id, sigc)
	defer func() {
		signal.Stop(sigc)
		close(sigc)
	}()

	condition := container.WaitConditionNextExit
	waitResultCh, waitErrCh := cli.ContainerWait(ctx, id, condition)

	err = cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	log.Debugf("started container %s", id)
	if os.Getenv("CIFUZZ_CONTAINER_SLEEP") != "" {
		log.Infof("Container %s is running.", id)
		log.Infof("Attach to it with: docker exec -it %s /bin/bash", id)
		log.Infof("Run the original command in the container with: eval $CMD")
		log.Infof("Press Ctrl+C to stop the container.")
	}

	resp, err := cli.ContainerAttach(ctx, id, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return errors.Wrap(err, "error attaching to container")
	}

	// Continuously print the container's stdout and stderr to the host's
	// stdout and stderr.
	go func() {
		defer resp.Close()

		_, err = stdcopy.StdCopy(outW, errW, resp.Reader)
		if err != nil {
			err := errors.Wrap(err, "error copying container logs")
			log.Error(err)
			return
		}
	}()

	// Wait for the result of the ContainerWait call.
	exitCode := 0
	select {
	case result := <-waitResultCh:
		if result.Error != nil {
			return errors.Errorf("error waiting for container: %v", result.Error.Message)
		} else {
			exitCode = int(result.StatusCode)
		}
	case err := <-waitErrCh:
		return errors.Wrap(err, "error waiting for container")
	}

	if exitCode != 0 {
		return errors.Errorf("container exited with non-zero exit code: %d", exitCode)
	}

	return nil
}
