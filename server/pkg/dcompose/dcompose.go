package dcompose

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// DockerComposeEngine docker compose config
type DockerComposeEngine struct {
	BinaryPath string
	Host       string
	YmlPath    string
}

func (engine *DockerComposeEngine) RunDockerComposeCommand(cmd string, args []string) (string, error) {
	if engine.BinaryPath == "" {
		engine.BinaryPath, _ = exec.LookPath("docker-compose")
	}

	joinArgs := strings.Join(args, " ")
	shell := engine.BinaryPath
	// TODO: suport other protocol
	if engine.Host != "" {
		splitString := strings.Split(engine.Host, "//")
		if len(splitString) == 0 {
			return "", errors.Errorf("invalid host format")
		}
		engine.Host = splitString[1]
		shell = fmt.Sprintf("%s -H %s", engine.BinaryPath, engine.Host)
	}
	if engine.YmlPath == "" {
		shell = fmt.Sprintf("%s %s %s", shell, cmd, joinArgs)
	} else {
		shell = fmt.Sprintf("%s -f %s %s %s", shell, engine.YmlPath, cmd, joinArgs)
	}
	var resp []byte
	var err error
	// TODO: support windows
	if resp, err = exec.Command("sh", "-c", shell).CombinedOutput(); err != nil {
		return string(resp), errors.Wrapf(err, "run exec command: %s", resp)
	}
	return string(resp), nil
}
