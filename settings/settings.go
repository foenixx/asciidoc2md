package settings

import "gopkg.in/yaml.v3"

type Headers2FileMap map[string]string

type Config struct {
	Headers map[string]Headers2FileMap//file.adoc -> id -> file.md
	CrossLinks map[string]string // maps adoc file name to its relative location: UserGuide.adoc -> ../user/
	UrlRewrites []Headers2FileMap // if link contains a specified key, then it's replaced with the provided value
	NavFile string
	InputFile string
	ArtifactsDir string
}

func Parse(data []byte) (*Config, error) {
	conf := Config{}
	err := yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

func (c *Config) String() (string, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
