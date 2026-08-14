package integration_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

type serviceSpec struct {
	name          string
	packagePath   string
	requiredKey   string
	requiredValue string
}

func TestBootstrapProcesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM process test is not supported on Windows")
	}

	root := repositoryRoot(t)
	services := []serviceSpec{
		{name: "gateway", packagePath: "./services/gateway/cmd/gateway", requiredKey: "HTTP_ADDR", requiredValue: ":8080"},
		{name: "identity", packagePath: "./services/identity/cmd/identity", requiredKey: "GRPC_ADDR", requiredValue: ":9091"},
		{name: "booking", packagePath: "./services/booking/cmd/booking", requiredKey: "GRPC_ADDR", requiredValue: ":9092"},
		{name: "notification", packagePath: "./services/notification/cmd/notification", requiredKey: "GRPC_ADDR", requiredValue: ":9093"},
	}

	for _, service := range services {
		t.Run(service.name, func(t *testing.T) {
			binary := buildService(t, root, service)
			assertMissingConfigFails(t, binary, service)
			assertGracefulSIGTERM(t, binary, service)
		})
	}
}

func buildService(t *testing.T, root string, service serviceSpec) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), service.name)
	cmd := exec.Command("go", "build", "-trimpath", "-o", binary, service.packagePath)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", service.name, err, output)
	}
	return binary
}

func assertMissingConfigFails(t *testing.T, binary string, service serviceSpec) {
	t.Helper()
	cmd := exec.Command(binary)
	cmd.Env = withEnvironment(map[string]string{
		"SEATFLOW_ENV":      "test",
		service.requiredKey: "",
	})
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("%s started without %s", service.name, service.requiredKey)
	}
	logOutput := string(output)
	if !strings.Contains(logOutput, service.requiredKey) ||
		!strings.Contains(logOutput, "startup failed") ||
		!strings.Contains(logOutput, `"environment":"unknown"`) {
		t.Fatalf("%s missing config output = %q, want clear %s startup error with unknown environment", service.name, output, service.requiredKey)
	}
}

func assertGracefulSIGTERM(t *testing.T, binary string, service serviceSpec) {
	t.Helper()
	cmd := exec.Command(binary)
	cmd.Env = withEnvironment(map[string]string{
		"SEATFLOW_ENV":      "test",
		service.requiredKey: service.requiredValue,
	})

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("%s stdout pipe: %v", service.name, err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", service.name, err)
	}

	logs := make(chan string, 8)
	scanDone := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var record struct {
				Message string `json:"msg"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				scanDone <- fmt.Errorf("decode log record: %w", err)
				close(logs)
				return
			}
			logs <- record.Message
		}
		scanDone <- scanner.Err()
		close(logs)
	}()

	waitResult := make(chan error, 1)
	go func() {
		waitResult <- cmd.Wait()
		close(waitResult)
	}()
	t.Cleanup(func() {
		select {
		case <-waitResult:
		default:
			_ = cmd.Process.Kill()
			<-waitResult
		}
	})

	waitForLog(t, logs, "service started", service.name)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal %s: %v", service.name, err)
	}
	waitForLog(t, logs, "shutdown completed", service.name)

	select {
	case err := <-waitResult:
		if err != nil {
			t.Fatalf("wait %s: %v; stderr: %s", service.name, err, stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not exit after SIGTERM", service.name)
	}

	if err := <-scanDone; err != nil {
		t.Fatalf("scan %s logs: %v", service.name, err)
	}
}

func waitForLog(t *testing.T, logs <-chan string, want, service string) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case message, ok := <-logs:
			if !ok {
				t.Fatalf("%s logs closed before %q", service, want)
			}
			if message == want {
				return
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for %s log %q", service, want)
		}
	}
}

func withEnvironment(overrides map[string]string) []string {
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if _, overridden := overrides[name]; !overridden {
			environment = append(environment, entry)
		}
	}
	for name, value := range overrides {
		if value != "" {
			environment = append(environment, name+"="+value)
		}
	}
	return environment
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}
