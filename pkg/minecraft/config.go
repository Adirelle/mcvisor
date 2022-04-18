package minecraft

import (
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultServerJar        = "server.jar"
	DefaultServerProperties = "server.properties"
	JaveHomeEnvName         = "JAVA_HOME"
)

type Config struct {
	BaseDir           string        `json:"-"`
	WorkingDir        string        `json:"working_dir,omitempty"`
	JavaHome          string        `json:"java_home,omitempty"`
	JavaParameters    []string      `json:"java_parameters"`
	ServerJar         string        `json:"server_jar,omitempty"`
	ServerProperties  string        `json:"server_properties,omitempty"`
	Parameters        []string      `json:"parameters"`
	ServerHost        string        `json:"server_host" validate:"ip|hostname|fqdn"`
	ServerPort        uint16        `json:"server_port"`
	PingPeriod        time.Duration `json:"ping_interval"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	ResponseTimeout   time.Duration `json:"response_timeout"`
}

func NewConfig(baseDir string) *Config {
	baseDir = filepath.Clean(baseDir)
	return &Config{
		BaseDir:    baseDir,
		WorkingDir: baseDir,
		JavaHome:   os.Getenv(JaveHomeEnvName),
		JavaParameters: []string{
			"-XX:+UnlockExperimentalVMOptions",
			"-XX:+UseG1GC",
			"-XX:G1NewSizePercent=20",
			"-XX:G1ReservePercent=20",
			"-XX:MaxGCPauseMillis=50",
			"-XX:G1HeapRegionSize=32M",
		},
		ServerJar:         DefaultServerJar,
		ServerProperties:  DefaultServerProperties,
		Parameters:        []string{"--nogui"},
		ServerHost:        "localhost",
		ServerPort:        25565,
		PingPeriod:        10 * time.Second,
		ConnectionTimeout: 5 * time.Second,
		ResponseTimeout:   5 * time.Second,
	}
}

func (c Config) AbsWorkingDir() string {
	return absPath(c.BaseDir, c.WorkingDir)
}

func (c Config) AbsJavaHome() string {
	return absPath(c.BaseDir, c.JavaHome)
}

func (c Config) Command() string {
	return filepath.Join(c.AbsJavaHome(), JavaCmd)
}

func (c Config) Arguments() []string {
	args := c.JavaParameters[:]
	args = append(args, "-jar", c.ServerJar)
	return append(args, c.Parameters...)
}

func (c Config) Env() []string {
	return append(
		os.Environ(),
		JaveHomeEnvName+"="+c.AbsJavaHome(),
	)
}

func (c Config) AbsServerProperties() string {
	return absPath(c.AbsWorkingDir(), c.ServerProperties)
}

func absPath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}
