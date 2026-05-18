package man

import (
	"strings"
	"testing"
)

func TestGenerateManPage(t *testing.T) {
	version := "1.7.1"
	page := GenerateManPage(version)
	if !strings.Contains(page, ".TH fz") {
		t.Error("missing .TH header")
	}
	if !strings.Contains(page, version) {
		t.Errorf("missing version %s", version)
	}
	if !strings.Contains(page, ".SH NAME") {
		t.Error("missing NAME section")
	}
	if !strings.Contains(page, ".SH SYNOPSIS") {
		t.Error("missing SYNOPSIS section")
	}
	if !strings.Contains(page, ".SH OPTIONS") {
		t.Error("missing OPTIONS section")
	}
}
