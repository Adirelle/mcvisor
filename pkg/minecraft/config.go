package minecraft

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultServerJar        = "server.jar"
	DefaultServerProperties = "server.properties"
	DefaultLog4JConf        = "mcvisor_log4J.xml"
	JaveHomeEnvName         = "JAVA_HOME"
)

type Config struct {
	Server *ServerConfig `json:"server"`
	Java   *JavaConfig   `json:"java"`
}

type ServerConfig struct {
	BaseDir    string         `json:"-"`
	WorkingDir string         `json:"working_dir,omitempty"`
	Jar        string         `json:"jar,omitempty"`
	Properties string         `json:"properties,omitempty"`
	Log4JConf  string         `json:"log4jxml,omitempty"`
	Options    []string       `json:"options"`
	Network    *NetworkConfig `json:"network"`
}

type NetworkConfig struct {
	Host              string        `json:"host,omitempty" validate:"omitempty,ip|hostname|fqdn"`
	Port              uint16        `json:"port,omitempty"`
	PingPeriod        time.Duration `json:"ping_interval"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	ResponseTimeout   time.Duration `json:"response_timeout"`
}

type JavaConfig struct {
	Home    string   `json:"home,omitempty" validate:"dir"`
	Options []string `json:"options"`
}

func NewConfig(baseDir string) *Config {
	baseDir = filepath.Clean(baseDir)
	return &Config{
		Java: &JavaConfig{
			Home: os.Getenv(JaveHomeEnvName),
			Options: []string{
				"-XX:+UnlockExperimentalVMOptions",
				"-XX:+UseG1GC",
				"-XX:G1NewSizePercent=20",
				"-XX:G1ReservePercent=20",
				"-XX:MaxGCPauseMillis=50",
				"-XX:G1HeapRegionSize=32M",
			},
		},
		Server: &ServerConfig{
			BaseDir:    baseDir,
			WorkingDir: baseDir,
			Jar:        DefaultServerJar,
			Properties: DefaultServerProperties,
			Log4JConf:  DefaultLog4JConf,
			Options:    []string{"--nogui"},
			Network: &NetworkConfig{
				PingPeriod:        10 * time.Second,
				ConnectionTimeout: 5 * time.Second,
				ResponseTimeout:   5 * time.Second,
			},
		},
	}
}

func (c Config) Command() []string {
	return append(c.Java.Command(), c.Server.Command()...)
}

func (c Config) Env() []string {
	return []string{JaveHomeEnvName + "=" + c.Java.Home}
}

func (c Config) WorkingDir() string {
	return c.Server.AbsWorkingDir()
}

func (c JavaConfig) AbsJavaCommand() string {
	return filepath.Join(c.Home, JavaCmd)
}

func (c JavaConfig) Command() []string {
	return append([]string{c.AbsJavaCommand()}, c.Options...)
}

func (c ServerConfig) AbsWorkingDir() string {
	return absPath(c.BaseDir, c.WorkingDir)
}

func (c ServerConfig) AbsLog4JConf() string {
	return absPath(c.AbsWorkingDir(), c.Log4JConf)
}

func (c ServerConfig) AbsServerProperties() string {
	return absPath(c.AbsWorkingDir(), c.Properties)
}

func (c ServerConfig) Command() []string {
	return append(
		[]string{
			fmt.Sprintf("-Dlog4j.configurationFile=%s", c.Log4JConf),
			"-jar",
			c.Jar,
		},
		c.Options...,
	)
}

func absPath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}
