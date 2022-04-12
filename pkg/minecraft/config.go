package minecraft

import (
	"os"
	"path/filepath"
)

type Config struct {
	WorkingDir string
	JavaHome string
	ServerJarPath string
	Parameters []string
	PIDPath string
}

func (c *Config) ResolvePath(baseDir string) {
	if c.JavaHome == "" {
		c.JavaHome = os.Getenv("JAVA_HOME")
	}
	if c.ServerJarPath == "" {
		c.ServerJarPath = "server.jar"
	}
	if c.PIDPath == "" {
		c.PIDPath = "server.pid"
	}

	c.WorkingDir = resolvePath(baseDir, c.WorkingDir)
	c.JavaHome = resolvePath(baseDir, c.JavaHome)
	c.ServerJarPath = resolvePath(baseDir, c.ServerJarPath)
	c.PIDPath = resolvePath(baseDir, c.PIDPath)
}

func (c Config) Command() string {
	return filepath.Join(c.JavaHome, "bin/java.exe")
}

func (c Config) CmdLine() []string {
	return append([]string{c.Command(), "-jar", c.ServerJarPath}, c.Parameters...)
}

func (c Config) Env() []string {
	return append(
		os.Environ(),
		"JAVA_HOME=" + c.JavaHome,
	)
}

func resolvePath(base, path string) string {
	if (filepath.IsAbs(path)) {
		return path
	}
	return filepath.Join(base, path)
}
