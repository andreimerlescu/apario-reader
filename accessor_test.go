package main

import (
	`testing`

	`github.com/stretchr/testify/assert`
)

func TestParseDDVersion(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name string
		args args
		want *DDVersion
	}{
		{
			name: "test v0.0.1",
			args: args{
				version: "v0.0.1",
			},
			want: &DDVersion{
				Major: 0,
				Minor: 0,
				Patch: 1,
			},
		},
		{
			name: "test v1.1.1",
			args: args{
				version: "v1.1.1",
			},
			want: &DDVersion{
				Major: 1,
				Minor: 1,
				Patch: 1,
			},
		},
		{
			name: "test v1.7.369",
			args: args{
				version: "v1.7.369",
			},
			want: &DDVersion{
				Major: 1,
				Minor: 7,
				Patch: 369,
			},
		},
		{
			name: "test v3.3.3",
			args: args{
				version: "v3.3.3",
			},
			want: &DDVersion{
				Major: 3,
				Minor: 3,
				Patch: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParseDDVersion(tt.args.version)
			assert.Equalf(t, tt.want, got, "ParseDDVersion(%v)", tt.args.version)
		})
	}
}
