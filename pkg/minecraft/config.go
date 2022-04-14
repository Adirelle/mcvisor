package minecraft

import (
	"os"
	"path/filepath"
)

const (
	DefaultServerJar        = "server.jar"
	DefaultPidfile          = "server.pid"
	DefaultServerProperties = "server.properties"
	JaveHomeEnvName         = "JAVA_HOME"
)

type Config struct {
	WorkingDir     string   `json:"working_dir,omitempty" validate:"dir"`
	JavaHome       string   `json:"java_home,omitempty" validate:"dir"`
	JavaParameters []string `json:"java_parameters,omitempty"`
	ServerJarPath  string   `json:"server_jar,omitempty" validate:"file"`
	Parameters     []string `json:"parameters,omitempty"`
	PidFile        string   `json:"pid_file,omitempty"`
}

func (c *Config) ConfigureDefaults() {
	if c.JavaHome == "" {
		c.JavaHome = os.Getenv(JaveHomeEnvName)
	}
	if c.JavaParameters == nil {
		c.JavaParameters = []string{
			"-XX:+UnlockExperimentalVMOptions",
			"-XX:+UseG1GC",
			"-XX:G1NewSizePercent=20",
			"-XX:G1ReservePercent=20",
			"-XX:MaxGCPauseMillis=50",
			"-XX:G1HeapRegionSize=32M",
		}
	}
	if c.ServerJarPath == "" {
		c.ServerJarPath = DefaultServerJar
	}
	if c.PidFile == "" {
		c.PidFile = DefaultPidfile
	}
}

func (c *Config) SetBaseDir(baseDir string) {
	c.WorkingDir = resolvePath(baseDir, c.WorkingDir)
	c.JavaHome = resolvePath(c.WorkingDir, c.JavaHome)
	c.ServerJarPath = resolvePath(c.WorkingDir, c.ServerJarPath)
	c.PidFile = resolvePath(c.WorkingDir, c.PidFile)
}

func (c Config) Command() string {
	return filepath.Join(c.JavaHome, JavaCmd)
}

func (c Config) CmdLine() []string {
	return append(
		append(
			append(
				[]string{c.Command()},
				c.JavaParameters...,
			),
			"-jar",
			c.ServerJarPath,
		),
		c.Parameters...,
	)
}

func (c Config) ServerPropertiesPath() string {
	return filepath.Join(c.WorkingDir, DefaultServerProperties)
}

func (c Config) Env() []string {
	return append(
		os.Environ(),
		JaveHomeEnvName+"="+c.JavaHome,
	)
}

func resolvePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}
