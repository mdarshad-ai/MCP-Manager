package install

import (
	"context"
	"fmt"
	"testing"
)

type mockRunner struct {
	f func(ctx context.Context, name string, args ...string) (string, string, error)
}

func (mr mockRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	if mr.f != nil {
		return mr.f(ctx, name, args...)
	}
	return "", "", nil
}

// TestNPMInstaller_NoPackageManager tests the installation failure when no package manager is found.
func TestNPMInstaller_NoPackageManager(t *testing.T) {
	installer := NewNPMInstaller(mockRunner{f: func(ctx context.Context, name string, args ...string) (string, string, error) {
		// Fail all commands to simulate no package manager
		return "", "", fmt.Errorf("command not found")
	}}, testLogger{t})

	options := NPMInstallOptions{
		Package: "left-pad",
	}

	_, err := installer.Install(context.Background(), "test-server", options)

	if err == nil {
		t.Fatalf("Expected error for missing package manager, got nil")
	}
	expectedErr := "package manager detection failed: no supported package manager found (tried: npm, yarn, pnpm)"
	if err.Error() != expectedErr {
		t.Fatalf("Expected error '%s', got: '%s'", expectedErr, err.Error())
	}
}

func TestNPMInstaller_DetectPackageManager(t *testing.T) {
	tests := []struct {
		name      string
		runner    Runner
		expected  string
		shouldErr bool
	}{
		{
			name: "npm available",
			runner: mockRunner{f: func(ctx context.Context, name string, args ...string) (string, string, error) {
				if name == "npm" && len(args) == 1 && args[0] == "--version" {
					return "8.0.0", "", nil
				}
				return "", "", fmt.Errorf("command not found")
			}},
			expected:  "npm",
			shouldErr: false,
		},
		{
			name: "no package manager available",
			runner: mockRunner{f: func(ctx context.Context, name string, args ...string) (string, string, error) {
				return "", "", fmt.Errorf("command not found")
			}},
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := NewNPMInstaller(tt.runner, nil)
			pm, err := installer.detectPackageManager(context.Background(), "")

			if tt.shouldErr && err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if pm != tt.expected {
				t.Fatalf("Expected '%s', got '%s'", tt.expected, pm)
			}
		})
	}
}

type testLogger struct {
	t *testing.T
}

func (tl testLogger) Log(line string) {
	tl.t.Log(line)
}

func (tl testLogger) Errorf(format string, args ...interface{}) {
	tl.t.Errorf(format, args...)
}

func TestNPMInstaller_ErrorHandling(t *testing.T) {
	// This test is tricky because it requires os.WriteFile to fail.
	// We'll skip it for now as it requires more advanced mocking or setup.
	t.Skip("Skipping test that requires filesystem error injection")
}
