package filesystem

import (
	"errors"
	"io"
	"os"
)

const (
	DirectoryPermission = os.FileMode(0o755) // rwxr-xr-x
	FilePermission      = os.FileMode(0o644) // rw-r--r--
)

func CopyFile(source string, destination string) (retErr error) {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := sourceFile.Close(); cerr != nil {
			retErr = errors.Join(retErr, errors.New("failed to close source file: "+cerr.Error()))
		}
	}()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := destinationFile.Close(); cerr != nil {
			retErr = errors.Join(retErr, errors.New("failed to close destination file: "+cerr.Error()))
		}
	}()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	if err := destinationFile.Chmod(FilePermission); err != nil {
		return err
	}

	return nil
}

func ReadFileAsString(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
