package settings

import "gopkg.in/yaml.v3"

type Headers2FileMap map[string]string

type Config struct {
	//file.adoc -> id -> file.md
	Headers map[string]Headers2FileMap `yaml:"headers"`
	// maps adoc file name to its relative location: UserGuide.adoc -> ../user/
	CrossLinks map[string]string `yaml:"cross_links"`
	// use alternative idmaps if ID isn't found
	IdMapFallbacks map[string]string `yaml:"idmap_fallbacks"`
	// if link contains a specified key, then it's replaced with the provided value
	UrlRewrites []Headers2FileMap `yaml:"url_rewrites"`
	NavFile string `yaml:"-"`
	InputFile string `yaml:"-"`
	ArtifactsDir string `yaml:"-"`
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
