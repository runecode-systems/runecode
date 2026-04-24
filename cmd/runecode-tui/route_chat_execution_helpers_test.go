package main

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestChatExecutionTerminal(t *testing.T) {
	tests := []struct {
		name string
		exec brokerapi.SessionTurnExecution
		want bool
	}{
		{
			name: "terminal when terminal outcome is cancelled",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState:  "waiting",
				TerminalOutcome: "cancelled",
			},
			want: true,
		},
		{
			name: "non terminal for waiting execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "waiting",
			},
			want: false,
		},
		{
			name: "non terminal for blocked execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "blocked",
			},
			want: false,
		},
		{
			name: "terminal for completed execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "completed",
			},
			want: true,
		},
		{
			name: "terminal for failed execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "failed",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatExecutionTerminal(tt.exec); got != tt.want {
				t.Fatalf("chatExecutionTerminal(%+v) = %t, want %t", tt.exec, got, tt.want)
			}
		})
	}
}
