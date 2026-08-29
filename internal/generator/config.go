package generator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	InputPath         string
	OutputDir         string
	Lang              string // ts|js
	SDKName           string
	BaseURLVar        string // exported base url var name
	AuthMode          string // none|bearer
	TokenFn           string // getToken
	AllowExternalRefs bool   `yaml:"allowExternalRefs"`
	EmitSchemas       bool   `yaml:"emitSchemas"`
	EmitOperations    bool   `yaml:"emitOperations"`
	Transport         string `yaml:"transport"`
	PackageName       string `yaml:"packageName"`
	GeneratorVersion  string `yaml:"generatorVersion"`
}

type FileConfig struct {
	InputPath         string `yaml:"input"`
	OutputDir         string `yaml:"output"`
	Lang              string `yaml:"language"`
	SDKName           string `yaml:"name"`
	BaseURLVar        string `yaml:"baseUrlVar"`
	AuthMode          string `yaml:"auth"`
	TokenFn           string `yaml:"tokenFn"`
	AllowExternalRefs bool   `yaml:"allowExternalRefs"`
	EmitSchemas       *bool  `yaml:"emitSchemas"`
	EmitOperations    *bool  `yaml:"emitOperations"`
	Transport         string `yaml:"transport"`
	PackageName       string `yaml:"packageName"`
	GeneratorVersion  string `yaml:"generatorVersion"`
}

func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var f FileConfig
	if err := yaml.Unmarshal(b, &f); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	c := Config{InputPath: f.InputPath, OutputDir: f.OutputDir, Lang: f.Lang, SDKName: f.SDKName, BaseURLVar: f.BaseURLVar, AuthMode: f.AuthMode, TokenFn: f.TokenFn, AllowExternalRefs: f.AllowExternalRefs, Transport: f.Transport, PackageName: f.PackageName, GeneratorVersion: f.GeneratorVersion, EmitSchemas: true, EmitOperations: true}
	if f.EmitSchemas != nil {
		c.EmitSchemas = *f.EmitSchemas
	}
	if f.EmitOperations != nil {
		c.EmitOperations = *f.EmitOperations
	}
	return c, nil
}
