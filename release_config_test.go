package workflowpluginadmin_test

import (
	"os"
	"strings"
	"testing"
)

func TestGoReleaserArchiveNamesMatchPluginInstaller(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yaml")
	if err != nil {
		t.Fatalf("read .goreleaser.yaml: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `name_template: "{{ .ProjectName }}-{{ .Os }}-{{ .Arch }}"`) {
		t.Fatal("GoReleaser archive names must use hyphens: workflow-plugin-admin-<os>-<arch>.tar.gz")
	}
	if strings.Contains(text, `name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"`) {
		t.Fatal("underscore archive names are not installable by wfctl plugin install")
	}
}
