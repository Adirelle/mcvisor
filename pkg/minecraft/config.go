package minecraft

import (
	"os"
	"path/filepath"
)

const (
	DefaultServerJar        = "server.jar"
	DefaultServerProperties = "server.properties"
	JaveHomeEnvName         = "JAVA_HOME"
)

type Config struct {
	BaseDir          string   `json:"-"`
	WorkingDir       string   `json:"working_dir,omitempty"`
	JavaHome         string   `json:"java_home,omitempty"`
	JavaParameters   []string `json:"java_parameters,omitempty"`
	ServerJar        string   `json:"server_jar,omitempty"`
	ServerProperties string   `json:"server_properties,omitempty"`
	Parameters       []string `json:"parameters,omitempty"`
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
		ServerJar:        DefaultServerJar,
		ServerProperties: DefaultServerProperties,
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
