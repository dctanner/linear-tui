package agents

import (
	"strings"
	"testing"
)

// TestParseCommand_SimpleCommand verifies a basic command is split correctly.
func TestParseCommand_SimpleCommand(t *testing.T) {
	// Use "echo" as a known binary
	binary, args, err := ParseCommand("echo {prompt}", "hello world", "")
	if err != nil {
		t.Fatalf("ParseCommand() error: %v", err)
	}
	if !strings.HasSuffix(binary, "echo") && !strings.Contains(binary, "echo") {
		t.Fatalf("binary = %q, want contains 'echo'", binary)
	}
	// args[0] = resolved binary, args[1] = prompt (single arg)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[1] != "hello world" {
		t.Fatalf("args[1] = %q, want %q", args[1], "hello world")
	}
}

// TestParseCommand_WithFlags verifies flags are preserved.
func TestParseCommand_WithFlags(t *testing.T) {
	_, args, err := ParseCommand("echo --flag1 --flag2 {prompt}", "the prompt", "")
	if err != nil {
		t.Fatalf("ParseCommand() error: %v", err)
	}
	// args: [binary, --flag1, --flag2, "the prompt"]
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[1] != "--flag1" {
		t.Fatalf("args[1] = %q, want %q", args[1], "--flag1")
	}
	if args[2] != "--flag2" {
		t.Fatalf("args[2] = %q, want %q", args[2], "--flag2")
	}
	if args[3] != "the prompt" {
		t.Fatalf("args[3] = %q, want %q", args[3], "the prompt")
	}
}

// TestParseCommand_BinaryNotFound verifies error when binary is missing.
func TestParseCommand_BinaryNotFound(t *testing.T) {
	_, _, err := ParseCommand("nonexistent-binary-xyz {prompt}", "test", "")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in error: %v", err)
	}
}

// TestParseCommand_EmptyTemplate verifies error on empty template.
func TestParseCommand_EmptyTemplate(t *testing.T) {
	_, _, err := ParseCommand("", "test", "")
	if err == nil {
		t.Fatal("expected error for empty template")
	}
}

// TestParseCommand_PromptIsSingleArg verifies {prompt} becomes one argument.
func TestParseCommand_PromptIsSingleArg(t *testing.T) {
	_, args, err := ParseCommand("echo {prompt}", "my prompt text", "")
	if err != nil {
		t.Fatalf("ParseCommand() error: %v", err)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[1] != "my prompt text" {
		t.Fatalf("args[1] = %q, want %q", args[1], "my prompt text")
	}
}

// TestParseCommand_MultilinePrompt verifies multiline prompts are preserved as a single arg.
func TestParseCommand_MultilinePrompt(t *testing.T) {
	prompt := "line one\nline two\nline three"
	_, args, err := ParseCommand("echo {prompt}", prompt, "")
	if err != nil {
		t.Fatalf("ParseCommand() error: %v", err)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if args[1] != prompt {
		t.Fatalf("args[1] = %q, want %q", args[1], prompt)
	}
}

// TestParseCommand_BranchReplacement verifies {branch} is replaced with branch name.
func TestParseCommand_BranchReplacement(t *testing.T) {
	_, args, err := ParseCommand("echo --branch {branch} {prompt}", "my prompt", "feature/my-branch")
	if err != nil {
		t.Fatalf("ParseCommand() error: %v", err)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[2] != "feature/my-branch" {
		t.Fatalf("args[2] = %q, want %q", args[2], "feature/my-branch")
	}
	if args[3] != "my prompt" {
		t.Fatalf("args[3] = %q, want %q", args[3], "my prompt")
	}
}

// TestParseCommand_BranchEmpty verifies empty {branch} becomes an empty arg.
func TestParseCommand_BranchEmpty(t *testing.T) {
	_, args, err := ParseCommand("echo --branch {branch} {prompt}", "my prompt", "")
	if err != nil {
		t.Fatalf("ParseCommand() error: %v", err)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[2] != "" {
		t.Fatalf("args[2] = %q, want empty string", args[2])
	}
}

// TestParseCommand_NoBranchPlaceholder verifies commands without {branch} work fine.
func TestParseCommand_NoBranchPlaceholder(t *testing.T) {
	_, args, err := ParseCommand("echo {prompt}", "hello", "some-branch")
	if err != nil {
		t.Fatalf("ParseCommand() error: %v", err)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %v", len(args), args)
	}
	if strings.Contains(args[1], "some-branch") {
		t.Fatalf("branch should not appear when no {branch} placeholder: %v", args)
	}
}
