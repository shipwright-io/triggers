package inventory

import (
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    string
		wantErr bool
	}{{
		name:    "http scheme URL",
		rawURL:  "https://github.com/username/repository.git",
		want:    "github.com/username/repository",
		wantErr: false,
	}, {
		name:    "git scheme URL",
		rawURL:  "git@github.com:username/repository.git",
		want:    "github.com/username/repository",
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareURLs(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{{
		name: "http scheme URLs",
		a:    "https://github.com/username/repository.git",
		b:    "http://github.com/username/repository",
		want: true,
	}, {
		name: "git and http URLs",
		a:    "https://github.com/username/repository.git",
		b:    "git@github.com:username/repository.git",
		want: true,
	}, {
		name: "http scheme different URLs",
		a:    "https://github.com/username/repository.git",
		b:    "https://github.com/username/another-repository.git",
		want: false,
	}, {
		name: "git and git schemes different URLs",
		a:    "https://github.com/username/repository.git",
		b:    "git@github.com:username/another-repository.git",
		want: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareURLs(tt.a, tt.b); got != tt.want {
				t.Errorf("CompareURLs() = %v, want %v", got, tt.want)
			}
		})
	}
}
