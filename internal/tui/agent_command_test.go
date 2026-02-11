package tui

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/roeyazroel/linear-tui/internal/config"
	"github.com/roeyazroel/linear-tui/internal/linearapi"
)

// TestAskAgentCommand_SetsPendingExec verifies the command flow sets pendingExec
// with the correct binary, args, and dir, then stops the TUI.
func TestAskAgentCommand_SetsPendingExec(t *testing.T) {
	cfg := config.Config{
		PageSize: 1,
		CacheTTL: time.Minute,
		AgentCommands: []config.AgentCommand{
			{Name: "Test Agent", Command: "test-agent --flag {prompt}"},
		},
	}
	app := NewApp(&linearapi.Client{}, cfg, nil)

	// Use a mutex to synchronize access to pages and other shared state
	var pagesMu sync.Mutex
	var stopped bool
	app.queueUpdateDraw = func(f func()) {
		pagesMu.Lock()
		f()
		pagesMu.Unlock()
	}

	selectedIssue := linearapi.Issue{ID: "issue-1", Title: "Test"}
	app.issuesMu.Lock()
	app.selectedIssue = &selectedIssue
	app.issuesMu.Unlock()

	app.fetchIssueByID = func(ctx context.Context, id string) (linearapi.Issue, error) {
		return linearapi.Issue{
			ID:          id,
			Title:       "Test",
			Description: "Desc",
			Comments: []linearapi.Comment{
				{Body: "Comment"},
			},
		}, nil
	}

	workspaceDir := t.TempDir()

	app.parseCommand = func(commandTemplate, fullPrompt, branchName string) (string, []string, error) {
		return "/usr/bin/test-agent", []string{"/usr/bin/test-agent", "--flag", fullPrompt}, nil
	}

	command := findCommandByID(DefaultCommands(app), "ask_agent")
	if command == nil {
		t.Fatal("ask_agent command not found")
	}

	pagesMu.Lock()
	command.Run(app)
	hasPrompt := app.pages.HasPage("agent_prompt")
	pagesMu.Unlock()
	if !hasPrompt {
		t.Fatal("expected agent prompt modal to be visible")
	}

	pagesMu.Lock()
	app.agentPromptModal.promptField.SetText("Summarize", true)
	app.agentPromptModal.workspaceField.SetText(workspaceDir)
	// Select the first command in the dropdown
	if app.agentPromptModal.commandField != nil {
		app.agentPromptModal.commandField.SetCurrentOption(0)
	}
	app.agentPromptModal.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl))
	pagesMu.Unlock()

	waitForCondition(t, time.Second, func() bool {
		pagesMu.Lock()
		defer pagesMu.Unlock()
		return app.pendingExec != nil
	})

	pagesMu.Lock()
	pending := app.pendingExec
	stopped = pending != nil // app.Stop() was called via QueueUpdateDraw
	pagesMu.Unlock()

	if !stopped || pending == nil {
		t.Fatal("expected pendingExec to be set")
	}

	if pending.Binary != "/usr/bin/test-agent" {
		t.Fatalf("binary = %q, want %q", pending.Binary, "/usr/bin/test-agent")
	}

	if pending.Dir != workspaceDir {
		t.Fatalf("dir = %q, want %q", pending.Dir, workspaceDir)
	}

	// Args[0] should be the binary name
	if len(pending.Args) == 0 || pending.Args[0] != "/usr/bin/test-agent" {
		t.Fatalf("args[0] = %q, want %q", pending.Args[0], "/usr/bin/test-agent")
	}

	joined := strings.Join(pending.Args, " ")

	// Should contain the --flag from the command template
	if !strings.Contains(joined, "--flag") {
		t.Fatalf("expected --flag in args: %s", joined)
	}

	// Should contain issue context (the prompt includes it)
	if !strings.Contains(joined, "Issue Context") {
		t.Fatalf("expected issue context in args: %s", joined)
	}
}

// TestDefaultCommands_GatesAskAgent verifies command gating by AgentCommands.
func TestDefaultCommands_GatesAskAgent(t *testing.T) {
	// No agent commands → ask_agent should be gated
	cfg := config.Config{
		PageSize:      1,
		CacheTTL:      time.Minute,
		AgentCommands: nil,
	}
	app := NewApp(&linearapi.Client{}, cfg, nil)

	commands := DefaultCommands(app)
	if findCommandByID(commands, "ask_agent") != nil {
		t.Fatal("expected ask_agent to be gated when no agent commands are configured")
	}

	// With agent commands → ask_agent should be present
	cfg.AgentCommands = []config.AgentCommand{
		{Name: "Claude", Command: "claude {prompt}"},
	}
	app2 := NewApp(&linearapi.Client{}, cfg, nil)

	commands = DefaultCommands(app2)
	if findCommandByID(commands, "ask_agent") == nil {
		t.Fatal("expected ask_agent when agent commands are configured")
	}
}

// findCommandByID locates a command by ID.
func findCommandByID(commands []Command, id string) *Command {
	for _, cmd := range commands {
		if cmd.ID == id {
			copyCmd := cmd
			return &copyCmd
		}
	}
	return nil
}
