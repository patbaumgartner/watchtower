package dockercompat

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{name: "minimum", version: "1.25", valid: true},
		{name: "newer", version: "1.44", valid: true},
		{name: "client maximum", version: "1.51", valid: true},
		{name: "below minimum", version: "1.24", valid: false},
		{name: "above client maximum", version: "1.52", valid: false},
		{name: "empty", version: "", valid: false},
		{name: "malformed", version: "latest", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.version)
			if test.valid && err != nil {
				t.Fatalf("Validate(%q) returned %v", test.version, err)
			}
			if !test.valid && err == nil {
				t.Fatalf("Validate(%q) succeeded", test.version)
			}
		})
	}
}

func TestConfiguredAPIVersion(t *testing.T) {
	t.Setenv("DOCKER_API_VERSION", "")
	if got := ConfiguredAPIVersion(); got != DefaultAPIVersion {
		t.Fatalf("ConfiguredAPIVersion() = %q, want %q", got, DefaultAPIVersion)
	}

	t.Setenv("DOCKER_API_VERSION", "1.44")
	if got := ConfiguredAPIVersion(); got != "1.44" {
		t.Fatalf("ConfiguredAPIVersion() = %q, want 1.44", got)
	}
}
