package settings

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSettings(t *testing.T) {
	map1 := map[string]string{"header 1": "file1.md","header 2": "file2.md"}
	map2 := map[string]string{"header 3": "file1.md","header 4": "file2.md"}
	conf := Config{
		Headers: map[string]IdMap {"UserGuide.adoc": map1, "InstallationGuide.adoc": map2},
		CrossLinks: map[string]string{ "UserGuide.adoc": "../user/", "InstallationGuide.adoc": "../installation/"},
	}
	str, _ := conf.String()
	t.Log("\n", str)
	//t.Fail()
}

func TestParseSettings(t *testing.T) {
	input := `
headers:
  header 1: file1.md
  header 2: file2.md
`
	conf, err := Parse([]byte(input))
	assert.NoError(t, err)
	t.Log(conf)
}