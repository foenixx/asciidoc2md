package settings

import "gopkg.in/yaml.v3"

type Config struct {
	Headers map[string]string
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
