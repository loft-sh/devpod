package kubernetes

import "testing"

func TestGetRegistryFromImageName(t *testing.T) {
	type args struct {
		imageName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "should return official docker registry",
			args: args{
				imageName: "docker.io/loftsh/devpod",
			},
			want:    OfficialDockerRegistry,
			wantErr: false,
		},
		{
			name: "should return official docker registry",
			args: args{
				imageName: "hub.docker.com/loftsh/devpod",
			},
			want:    OfficialDockerRegistry,
			wantErr: false,
		},
		{
			name: "should return official docker registry",
			args: args{
				imageName: "nginx:latest",
			},
			want:    OfficialDockerRegistry,
			wantErr: false,
		},
		{
			name: "should return github registry",
			args: args{
				imageName: "ghcr.io/loftsh/devpod",
			},
			want:    "ghcr.io",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRegistryFromImageName(tt.args.imageName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRegistryFromImageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetRegistryFromImageName() got = %v, want %v", got, tt.want)
			}
		})
	}
}
