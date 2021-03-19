package settings

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)


func TestSettings1(t *testing.T) {
	//map1 := map[string]string{"header 1": "file1.md","header 2": "file2.md"}
	//map2 := map[string]string{"header 3": "file1.md","header 4": "file2.md"}
	//conf := struct {
	//	UrlRewrites []string
	//}{[]string{ "htts://mytessa.ru/docs/test.html","test.adoc","routes.adoc","AdministratorGuide.adoc"}}
	conf := struct {
		UrlRewrites []map[string]string
	}{[]map[string]string{ {"htts://mytessa.ru/docs/test.html":"test.adoc", "test1":"tst2"},{"routes.adoc":"AdministratorGuide.adoc"}}}

	data, err := yaml.Marshal(&conf)
	assert.NoError(t, err)
	t.Log("\n", string(data))
	//t.Fail()
}


func TestSettings(t *testing.T) {
	map1 := map[string]string{"header 1": "file1.md","header 2": "file2.md"}
	map2 := map[string]string{"header 3": "file1.md","header 4": "file2.md"}
	conf := Config{
		Headers: map[string]IdMap {"UserGuide.adoc": map1, "InstallationGuide.adoc": map2},
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
urlrewrites:
  - rule1: value1
  - rule2: value2
`
	conf, err := Parse([]byte(input))
	assert.NoError(t, err)
	t.Log(conf)
}