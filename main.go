package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

const (
	// log prefix
	app = "fargate-sidecar-datadog-agent"

	// Container to ignore the check
	ignoreContainerName = "^~internal~.*"
)

var (
	// ECSMetaDataURL ... This URL is for getting ECS task information
	ECSMetaDataURL = os.ExpandEnv("${ECS_CONTAINER_METADATA_URI}/task")
)

// EcsTask ... Store ECS state.
type EcsTask struct {
	Containers []*Container
}

// Container ... Store Container state.
type Container struct {
	Name        string
	KnownStatus string
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix(app + ": ")
	flag.Parse()

	if err := Run(); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

// Run ... Run fargate-sidecar-datadog-agent.
func Run() error {
	cmd, err := ExecCmd(flag.Args())
	if err != nil {
		return err
	}

	// set channel, syscall
	sigCh := make(chan os.Signal, 1)
	doneCh := make(chan struct{})
	errCh := make(chan error)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go cmd.Wait()
	go CheckStopContainer(errCh, doneCh)
	go SignalHandler(cmd, sigCh, doneCh)

	for {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			return nil
		}
	}
}

// ExecCmd ...  Execute a command.
func ExecCmd(args []string) (*exec.Cmd, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// CheckStopContainer ... Check every second if the container is stopped.
func CheckStopContainer(errCh chan error, doneCh chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			task, err := GetEcsTask()
			if err != nil {
				errCh <- err
			}

			isStopped := true
			for _, c := range task.Containers {
				re := regexp.MustCompile(ignoreContainerName)
				if !re.MatchString(c.Name) && c.KnownStatus != "STOPPED" {
					isStopped = false
					break
				}
			}

			if isStopped {
				doneCh <- struct{}{}
			}
		}
	}

}

// GetEcsTask ... Get all ecs task.
func GetEcsTask() (*EcsTask, error) {
	resp, err := http.Get(ECSMetaDataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var task EcsTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

// SignalHandler ... Receive signal handler.
func SignalHandler(cmd *exec.Cmd, sigCh chan os.Signal, doneCh chan struct{}) {
	for {
		select {
		case <-sigCh:
			log.Println("shutdown after 5 second...")
			time.Sleep(5 * time.Second)

			doneCh <- struct{}{}
		}
	}
}
