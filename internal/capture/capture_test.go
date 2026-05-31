package capture

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestInteractiveCapturerUsesScreencaptureArguments(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("interactive screencapture is macOS-only")
	}
	tempDir := t.TempDir()
	recordFile := filepath.Join(tempDir, "args.txt")
	scriptPath := filepath.Join(tempDir, "screencapture")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$RECORD_FILE"
last=""
for arg in "$@"; do
  last="$arg"
done
printf 'png' > "$last"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RECORD_FILE", recordFile)
	now := time.Date(2026, 5, 30, 19, 30, 0, 0, time.UTC)
	capturer := &InteractiveCapturer{
		Command: scriptPath,
		TempDir: tempDir,
		Now:     func() time.Time { return now },
	}

	got, err := capturer.CaptureChart(context.Background(), WindowTarget{})
	if err != nil {
		t.Fatalf("CaptureChart returned error: %v", err)
	}

	args, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"-i\n", "-x\n", "-t\npng\n"} {
		if !strings.Contains(string(args), fragment) {
			t.Fatalf("args = %q, want fragment %q", string(args), fragment)
		}
	}
	if got.Path == "" || !strings.HasPrefix(got.Path, tempDir) {
		t.Fatalf("path = %q, want temp png path", got.Path)
	}
	if got.MimeType != "image/png" || got.Source != "desktop_region" || !got.CapturedAt.Equal(now) {
		t.Fatalf("capture = %#v, want PNG desktop region metadata", got)
	}
	data, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "png" {
		t.Fatalf("captured bytes = %q, want png", string(data))
	}
}

func TestInteractiveCapturerCapturesSelectedWindow(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("window screencapture is macOS-only")
	}
	tempDir := t.TempDir()
	recordFile := filepath.Join(tempDir, "args.txt")
	scriptPath := filepath.Join(tempDir, "screencapture")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$RECORD_FILE"
last=""
for arg in "$@"; do
  last="$arg"
done
printf 'png' > "$last"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RECORD_FILE", recordFile)
	now := time.Date(2026, 5, 30, 19, 45, 0, 0, time.UTC)
	capturer := &InteractiveCapturer{
		Command: scriptPath,
		TempDir: tempDir,
		Now:     func() time.Time { return now },
		ListWindowsFunc: func(context.Context) ([]WindowInfo, error) {
			return []WindowInfo{
				{ID: 41, AppName: "Safari", Title: "News"},
				{ID: 42, AppName: "Trader Workstation", Title: "NVDA 5m"},
			}, nil
		},
	}

	got, err := capturer.CaptureChart(context.Background(), WindowTarget{AppName: "Trader Workstation", Title: "NVDA 5m"})
	if err != nil {
		t.Fatalf("CaptureChart returned error: %v", err)
	}

	args, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"-l\n42\n", "-x\n", "-t\npng\n"} {
		if !strings.Contains(string(args), fragment) {
			t.Fatalf("args = %q, want fragment %q", string(args), fragment)
		}
	}
	if strings.Contains(string(args), "-i\n") {
		t.Fatalf("args = %q, selected window capture must not use interactive selection", string(args))
	}
	if got.Source != "window:Trader Workstation - NVDA 5m" || !got.CapturedAt.Equal(now) {
		t.Fatalf("capture = %#v, want selected window metadata", got)
	}
}

func TestInteractiveCapturerFallsBackToWindowNameWhenStoredIDIsStale(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("window screencapture is macOS-only")
	}
	tempDir := t.TempDir()
	recordFile := filepath.Join(tempDir, "args.txt")
	scriptPath := filepath.Join(tempDir, "screencapture")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$RECORD_FILE"
last=""
for arg in "$@"; do
  last="$arg"
done
printf 'png' > "$last"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RECORD_FILE", recordFile)
	capturer := &InteractiveCapturer{
		Command: scriptPath,
		TempDir: tempDir,
		ListWindowsFunc: func(context.Context) ([]WindowInfo, error) {
			return []WindowInfo{
				{ID: 42, AppName: "Safari", Title: "Market Notes"},
				{ID: 99, AppName: "Trader Workstation", Title: "NVDA 5m"},
			}, nil
		},
	}

	_, err := capturer.CaptureChart(context.Background(), WindowTarget{ID: 42, AppName: "Trader Workstation", Title: "NVDA 5m"})
	if err != nil {
		t.Fatalf("CaptureChart returned error: %v", err)
	}

	args, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "-l\n99\n") {
		t.Fatalf("args = %q, want refreshed window id 99", string(args))
	}
}
