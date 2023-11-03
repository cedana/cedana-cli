package utils

import (
	"fmt"
	"os"
	"testing"
)

const CEDANA_ENV_ENV_VAR = "CEDANA_ENV"

func TestMain(m *testing.M) {
	originalEnv := os.Getenv(CEDANA_ENV_ENV_VAR)
	SetConfigFile("")
	code := m.Run()
	os.Setenv(CEDANA_ENV_ENV_VAR, originalEnv)
	os.Exit(code)
}

func TestCannotFindConfigFile(t *testing.T) {
	originalCedanaPath := GetCedanaPath()
	defer SetCedanaPath(originalCedanaPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}
	os.MkdirAll(cedanaTestPath(homeDir), os.ModePerm)
	SetCedanaPath(cedanaTestDir())

	_, err = InitCedanaConfig()
	expectedErrorMessage := fmt.Sprintf(
		"error loading config file: Config File \"cedana_config\" Not Found in \"[%s/.cedana_test]\". Make sure that config exists and that it's formatted correctly!",
		homeDir,
	)

	if err == nil {
		t.Fatal("expected an error when initializing cedana config")
	}

	if err.Error() != expectedErrorMessage {
		t.Errorf("unexpected error \"%s\" != \"%s\"", err, expectedErrorMessage)
	}
}

func TestCannotFindConfigFileInDevEnv(t *testing.T) {
	originalCedanaPath := GetCedanaPath()
	defer SetCedanaPath(originalCedanaPath)

	os.Setenv("CEDANA_ENV", "dev")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}

	os.MkdirAll(cedanaTestPath(homeDir), os.ModePerm)
	SetCedanaPath(cedanaTestDir())

	_, err = InitCedanaConfig()
	expectedErrorMessage := fmt.Sprintf(
		"error loading config file: Config File \"cedana_config_dev\" Not Found in \"[%s/.cedana_test]\". Make sure that config exists and that it's formatted correctly!",
		homeDir,
	)

	if err == nil {
		t.Fatal("expected an error when initializing cedana config")
	}

	if err.Error() != expectedErrorMessage {
		t.Errorf("unexpected error \"%s\" != \"%s\"", err, expectedErrorMessage)
	}
}

func TestOverrideConfigFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	SetConfigFile(fmt.Sprintf("%s/testresources/test_config.json", wd))
	_, err = InitCedanaConfig()
	if err != nil {
		t.Error(err)
	}
}

func TestOverrideConfigFileDoesNotExist(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	SetConfigFile(fmt.Sprintf("%s/testresources/non_exisitent.json", wd))
	_, err = InitCedanaConfig()
	expectedErrorMessage := fmt.Sprintf(
		"error loading config file: open %s/testresources/non_exisitent.json: no such file or directory. Make sure that config exists and that it's formatted correctly!", wd)

	if err.Error() != expectedErrorMessage {
		t.Errorf("unexpected error \"%s\" != \"%s\"", err, expectedErrorMessage)
	}
}

func cedanaTestDir() string {
	return ".cedana_test"
}

func cedanaTestPath(homeDir string) string {
	return fmt.Sprintf("%s/%s", homeDir, cedanaTestDir())
}
