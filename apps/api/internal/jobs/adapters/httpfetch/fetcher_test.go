package httpfetch

import (
	"net/url"
	"testing"
)

func TestValidateURLRejectsInternalTargets(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/image.png", "http://10.0.0.1/a.jpg", "file:///etc/passwd", "https://user:pass@example.com/a.png"} {
		parsed, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateURL(parsed); err == nil {
			t.Errorf("validateURL(%q) accepted internal or unsafe URL", raw)
		}
	}
}

func TestValidateURLAcceptsPublicHost(t *testing.T) {
	parsed, _ := url.Parse("https://images.example.com/photo.png")
	if err := validateURL(parsed); err != nil {
		t.Fatalf("validateURL() error = %v", err)
	}
}
