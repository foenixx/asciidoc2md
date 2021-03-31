package settings

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)


func debug1(t *testing.T) {
	map1 := map[string]string{"header 1": "file1.md","header 2": "file2.md"}
	map2 := map[string]string{"header 3": "file1.md","header 4": "file2.md"}
	conf := Config{
		Headers: map[string]Headers2FileMap {"UserGuide.adoc": map1, "InstallationGuide.adoc": map2},
		CrossLinks: map[string]string{ "UserGuide.adoc": "../user/", "InstallationGuide.adoc": "../installation/"},
		//UrlRewrites: map[string]string{ "htts://mytessa.ru/docs/test.html": "test.adoc", "routes.adoc": "AdministratorGuide.adoc"},
	}
	str, _ := conf.String()
	t.Log("\n", str)
	//t.Fail()
}

func TestParseSettings(t *testing.T) {
	input := `
headers:
  file1:
    header 1: file1.md
    header 2: file2.md
url_rewrites:
  - rule1: value1
  - rule2: value2
idmap_fallbacks:
  map1.adoc.idmap: map2.adoc.idmap
cross_links:
  file.adoc: relative/path
`
	conf, err := Parse([]byte(input))
	assert.NoError(t, err)
	//t.Logf("%+v", conf)
	data, err := yaml.Marshal(conf)
	assert.NoError(t, err)
	//t.Logf("%+v", string(data))
	conf2, err := Parse(data)
	assert.NoError(t, err)
	//conf2.CrossLinks["file.adoc"] = "sdf"
	//t.Logf("%+v", conf2)
	assert.Equal(t, conf, conf2)
}