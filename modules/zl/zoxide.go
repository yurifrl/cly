package zl

import (
	"os/exec"
	"strings"
)

// Function variables for testing
var (
	findZoxidePath           = exec.LookPath
	checkZoxideInstalledFunc = CheckZoxideInstalled
	execZoxideQuery          = execZoxideQueryReal
	execZoxideInteractive    = execZoxideInteractiveReal
	execZoxideAdd            = execZoxideAddReal
)

// CheckZoxideInstalled returns true if zoxide is in PATH
func CheckZoxideInstalled() bool {
	_, err := findZoxidePath("zoxide")
	return err == nil
}

// QueryZoxide queries zoxide for the best matching directory
func QueryZoxide(keywords ...string) (string, error) {
	if !checkZoxideInstalledFunc() {
		return "", nil
	}

	path, err := execZoxideQuery(keywords...)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(path), nil
}

// QueryZoxideInteractive launches interactive zoxide picker
func QueryZoxideInteractive() (string, error) {
	if !checkZoxideInstalledFunc() {
		return "", nil
	}

	path, err := execZoxideInteractive()
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(path), nil
}

// UpdateZoxide adds a directory to zoxide database
func UpdateZoxide(dir string) error {
	if !checkZoxideInstalledFunc() {
		return nil
	}

	return execZoxideAdd(dir)
}

// Real implementations
func execZoxideQueryReal(keywords ...string) (string, error) {
	args := append([]string{"query"}, keywords...)
	out, err := exec.Command("zoxide", args...).Output()
	return string(out), err
}

func execZoxideInteractiveReal() (string, error) {
	out, err := exec.Command("zoxide", "query", "-i").Output()
	return string(out), err
}

func execZoxideAddReal(dir string) error {
	return exec.Command("zoxide", "add", dir).Run()
}
