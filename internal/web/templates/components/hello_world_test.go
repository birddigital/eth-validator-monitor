package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelloWorld(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "renders greeting with name",
			input: "Alice",
			contains: []string{
				"Hello, Alice!",
				"Welcome to Ethereum Validator Monitor",
				`<div class="greeting">`,
			},
		},
		{
			name:  "handles empty name",
			input: "",
			contains: []string{
				"Hello, !",
				"Welcome to Ethereum Validator Monitor",
			},
		},
		{
			name:  "handles special characters",
			input: "Validator <Node>",
			contains: []string{
				"Validator &lt;Node&gt;",
				"Welcome to Ethereum Validator Monitor",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			component := HelloWorld(tt.input)
			buf := new(bytes.Buffer)
			ctx := context.Background()

			// Act
			err := component.Render(ctx, buf)

			// Assert
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			output := buf.String()
			for _, expectedText := range tt.contains {
				if !strings.Contains(output, expectedText) {
					t.Errorf("expected output to contain %q, got:\n%s", expectedText, output)
				}
			}
		})
	}
}
