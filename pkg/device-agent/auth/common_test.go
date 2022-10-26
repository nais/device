package auth

import (
	"testing"
)

func TestIsChrome(t *testing.T) {
	chromeUserAgent := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"
	firefoxUserAgent := "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:106.0) Gecko/20100101 Firefox/106.0"
	if !isChrome(chromeUserAgent) {
		t.Errorf("IsChrome(%q) should be true", chromeUserAgent)
	}

	if isChrome(firefoxUserAgent) {
		t.Errorf("IsChrome(%q) should be false", firefoxUserAgent)
	}
}
