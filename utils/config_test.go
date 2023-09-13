package utils

import (
	"testing"
	"os"
	"fmt"
)

const CEDANA_ENV_ENV_VAR = "CEDANA_ENV"

func TestMain(m *testing.M) {
	originalEnv := os.Getenv(CEDANA_ENV_ENV_VAR)
    code := m.Run() 
	os.Setenv(CEDANA_ENV_ENV_VAR, originalEnv)
    os.Exit(code)
}

func TestCannotFindConfigFile(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}
	
	_, err = InitCedanaConfig()
	expectedErrorMessage := fmt.Sprintf(
		"error loading config file: Config File \"cedana_config\" Not Found in \"[%s/.cedana]\". Make sure that config exists and that it's formatted correctly!",
		homeDir,
	)


	if err.Error() != expectedErrorMessage {
		t.Errorf("unexpected error \"%s\" != \"%s\"", err, expectedErrorMessage)
	}
}

func TestCannotFindConfigFileInDevEnv(t *testing.T) {
	os.Setenv("CEDANA_ENV", "dev")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}
	
	_, err = InitCedanaConfig()
	expectedErrorMessage := fmt.Sprintf(
		"error loading config file: Config File \"cedana_config_dev\" Not Found in \"[%s/.cedana]\". Make sure that config exists and that it's formatted correctly!",
		homeDir,
	)


	if err.Error() != expectedErrorMessage {
		t.Errorf("unexpected error \"%s\" != \"%s\"", err, expectedErrorMessage)
	}
}