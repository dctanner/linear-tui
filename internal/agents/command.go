package agents

import (
	"fmt"
	"os/exec"
	"strings"
)

// ParseCommand splits the command template into tokens, then substitutes {prompt}
// and {branch} placeholders as single arguments (preserving multiline content and
// whitespace). It resolves the binary via exec.LookPath and returns the absolute path.
func ParseCommand(commandTemplate, fullPrompt, branchName string) (binary string, args []string, err error) {
	tokens := strings.Fields(commandTemplate)
	if len(tokens) == 0 {
		return "", nil, fmt.Errorf("empty command template")
	}

	resolved, err := exec.LookPath(tokens[0])
	if err != nil {
		return "", nil, fmt.Errorf("binary %q not found: %w", tokens[0], err)
	}

	args = make([]string, 0, len(tokens))
	args = append(args, resolved)
	for _, tok := range tokens[1:] {
		switch tok {
		case "{prompt}":
			args = append(args, fullPrompt)
		case "{branch}":
			args = append(args, branchName)
		default:
			args = append(args, tok)
		}
	}

	return resolved, args, nil
}
