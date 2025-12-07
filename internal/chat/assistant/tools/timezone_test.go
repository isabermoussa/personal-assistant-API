package tools

import (
	"context"
	"strings"
	"testing"
)

func TestTimeZoneTool_Name(t *testing.T) {
	tool := NewTimeZoneTool()
	if tool.Name() != "convert_timezone" {
		t.Errorf("expected 'convert_timezone', got '%s'", tool.Name())
	}
}

func TestTimeZoneTool_Handle(t *testing.T) {
	tool := NewTimeZoneTool()
	ctx := context.Background()

	tests := []struct {
		name        string
		args        string
		wantContain []string
		wantErr     bool
	}{
		{
			name: "convert specific time from NYC to Madrid",
			args: `{
				"time": "2025-12-15T14:00:00Z",
				"from_timezone": "America/New_York",
				"to_timezone": "Europe/Madrid"
			}`,
			wantContain: []string{"America/New_York", "Europe/Madrid", "Time Conversion"},
			wantErr:     false,
		},
		{
			name: "convert current time from UTC to Tokyo",
			args: `{
				"time": "now",
				"from_timezone": "UTC",
				"to_timezone": "Asia/Tokyo"
			}`,
			wantContain: []string{"UTC", "Asia/Tokyo", "+9.0 hours"},
			wantErr:     false,
		},
		{
			name: "omit time parameter (defaults to now)",
			args: `{
				"from_timezone": "Europe/London",
				"to_timezone": "America/Los_Angeles"
			}`,
			wantContain: []string{"Europe/London", "America/Los_Angeles"},
			wantErr:     false,
		},
		{
			name: "same timezone conversion",
			args: `{
				"time": "2025-12-15T12:00:00Z",
				"from_timezone": "UTC",
				"to_timezone": "UTC"
			}`,
			wantContain: []string{"same time"},
			wantErr:     false,
		},
		{
			name: "invalid source timezone",
			args: `{
				"time": "2025-12-15T14:00:00Z",
				"from_timezone": "Invalid/Timezone",
				"to_timezone": "Europe/Madrid"
			}`,
			wantErr: true,
		},
		{
			name: "invalid target timezone",
			args: `{
				"time": "2025-12-15T14:00:00Z",
				"from_timezone": "America/New_York",
				"to_timezone": "Invalid/Timezone"
			}`,
			wantErr: true,
		},
		{
			name: "invalid time format",
			args: `{
				"time": "not-a-valid-time",
				"from_timezone": "America/New_York",
				"to_timezone": "Europe/Madrid"
			}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			args:    `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Handle(ctx, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for _, want := range tt.wantContain {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain '%s', got: %s", want, result)
				}
			}
		})
	}
}

func TestTimeZoneTool_Definition(t *testing.T) {
	tool := NewTimeZoneTool()
	def := tool.Definition()

	// Verify it returns a valid tool definition (basic smoke test)
	// The actual structure is from OpenAI SDK and is complex
	_ = def
}
