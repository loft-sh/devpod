package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveDevContainerJSON(t *testing.T) {
	type args struct {
		config *DevContainerConfig
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantJSON string
	}{
		{
			name: "test omit build field in devcontainer.json",
			args: args{
				config: &DevContainerConfig{
					ImageContainer: ImageContainer{
						Image: "test",
					},
				},
			},
			wantErr:  false,
			wantJSON: `{"image":"test"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp(os.TempDir(), "test-devcontainer")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			tt.args.config.Origin = filepath.Join(tmpDir, "devcontainer.json")

			if err := SaveDevContainerJSON(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("SaveDevContainerJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			contents, err := os.ReadFile(tt.args.config.Origin)
			if err != nil {
				t.Fatalf("Failed to read file contents: %v", err)
			}
			if string(contents) != tt.wantJSON {
				t.Errorf("Expected JSON = %v, got %v", tt.wantJSON, string(contents))
			}
		})
	}
}
