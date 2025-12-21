package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantDev bool
		wantErr bool
	}{
		{
			name:    "standard semver with v prefix",
			input:   "v0.2.5",
			want:    "0.2.5",
			wantDev: false,
			wantErr: false,
		},
		{
			name:    "semver without v prefix",
			input:   "0.2.5",
			want:    "0.2.5",
			wantDev: false,
			wantErr: false,
		},
		{
			name:    "dev build",
			input:   "dev",
			want:    "dev",
			wantDev: true,
			wantErr: false,
		},
		{
			name:    "semver with prerelease",
			input:   "v1.0.0-beta.1",
			want:    "1.0.0-beta.1",
			wantDev: false,
			wantErr: false,
		},
		{
			name:    "semver with build metadata",
			input:   "v1.0.0+build.123",
			want:    "1.0.0+build.123",
			wantDev: false,
			wantErr: false,
		},
		{
			name:    "invalid semver",
			input:   "not-a-version",
			want:    "",
			wantDev: false,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantDev: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.input, v.Raw)
			assert.Equal(t, tt.want, v.String())
			assert.Equal(t, tt.wantDev, v.IsDev)
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{
			name: "v1 older than v2",
			v1:   "v0.2.4",
			v2:   "v0.2.5",
			want: -1,
		},
		{
			name: "v1 equal to v2",
			v1:   "v0.2.5",
			v2:   "v0.2.5",
			want: 0,
		},
		{
			name: "v1 newer than v2",
			v1:   "v0.2.6",
			v2:   "v0.2.5",
			want: 1,
		},
		{
			name: "major version difference",
			v1:   "v1.0.0",
			v2:   "v0.2.5",
			want: 1,
		},
		{
			name: "prerelease vs release",
			v1:   "v1.0.0-beta.1",
			v2:   "v1.0.0",
			want: -1,
		},
		{
			name: "dev vs release",
			v1:   "dev",
			v2:   "v0.2.5",
			want: -1,
		},
		{
			name: "release vs dev",
			v1:   "v0.2.5",
			v2:   "dev",
			want: 1,
		},
		{
			name: "dev vs dev",
			v1:   "dev",
			v2:   "dev",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, err := ParseVersion(tt.v1)
			require.NoError(t, err)

			v2, err := ParseVersion(tt.v2)
			require.NoError(t, err)

			got := v1.Compare(v2)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionIsOlderThan(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    bool
	}{
		{
			name:    "current is older",
			current: "v0.2.4",
			target:  "v0.2.5",
			want:    true,
		},
		{
			name:    "current is same",
			current: "v0.2.5",
			target:  "v0.2.5",
			want:    false,
		},
		{
			name:    "current is newer",
			current: "v0.2.6",
			target:  "v0.2.5",
			want:    false,
		},
		{
			name:    "dev is always older",
			current: "dev",
			target:  "v0.2.5",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, err := ParseVersion(tt.current)
			require.NoError(t, err)

			target, err := ParseVersion(tt.target)
			require.NoError(t, err)

			got := current.IsOlderThan(target)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantDev bool
	}{
		{
			name:    "dev version",
			version: "dev",
			wantDev: true,
		},
		{
			name:    "valid semver",
			version: "v0.2.5",
			wantDev: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := GetCurrentVersion(tt.version)
			require.NotNil(t, v)
			assert.Equal(t, tt.wantDev, v.IsDev)

			if v.IsDev {
				assert.Equal(t, tt.version, v.Raw)
			} else {
				assert.NotEmpty(t, v.String())
			}
		})
	}
}
