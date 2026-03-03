package models

type BindResource struct {
	Binds []Bind `yaml:"binds"`
}

type Bind struct {
	Name        string    `yaml:"name"`
	Override    bool      `yaml:"override"`
	Enabled     bool      `yaml:"enabled"`
	Description string    `yaml:"description,omitempty"`
	Type        string    `yaml:"type"`
	IP          string    `yaml:"ip,omitempty"`
	Port        int       `yaml:"port"`
	Backend     Backend   `yaml:"backend,omitempty"`
	Redirect    *Redirect `yaml:"redirect,omitempty"`
}

type Backend struct {
	Servers []Server `yaml:"servers,omitempty"`
}

type Server struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

type Redirect struct {
	Scheme string `yaml:"scheme"`
	Port   int    `yaml:"port,omitempty"`
	Code   int    `yaml:"code,omitempty"`
}
