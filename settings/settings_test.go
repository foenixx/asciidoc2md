package settings

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSettings(t *testing.T) {
	conf := Config{Headers: map[string]string{"header 1": "file1.md","header 2": "file2.md"}}
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