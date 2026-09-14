package vercelutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/khanakia/vercelgate/pkg/jsonupdate"
	"github.com/khanakia/vercelgate/pkg/utils"

	"github.com/adrg/xdg"
)

var (
	authJsonFileName = "auth.json"
)

func SetAuthToken(token string) error {
	filePath, err := AuthJsonFile()
	if err != nil {
		return err
	}

	fileBytes, err := utils.OpenFile(filePath)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	jsonupd := jsonupdate.NewJsonUpdate(string(fileBytes))

	jsonupd.Set("token", token)

	err = os.WriteFile(filePath, []byte(jsonupd.String()), 0644)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func SetCurrentTeam(teamID string) error {
	filePath, err := ConfigJsonFile()
	if err != nil {
		return err
	}

	fileBytes, err := utils.OpenFile(filePath)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	jsonupd := jsonupdate.NewJsonUpdate(string(fileBytes))

	jsonupd.Set("currentTeam", teamID)

	err = os.WriteFile(filePath, []byte(jsonupd.Pretty()), 0644)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func DeleteCurrentTeam() error {
	filePath, err := ConfigJsonFile()
	if err != nil {
		return err
	}

	fileBytes, err := utils.OpenFile(filePath)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	jsonupd := jsonupdate.NewJsonUpdate(string(fileBytes))

	jsonupd.Deleete("currentTeam")

	err = os.WriteFile(filePath, []byte(jsonupd.String()), 0644)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func ParseAuthFile(path string) (*AuthConfig, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var authConfig AuthConfig
	err = json.Unmarshal(fileBytes, &authConfig)
	if err != nil {
		return nil, err
	}
	return &authConfig, nil
}

type AuthConfig struct {
	Token string `json:"token"`
}

func AuthJsonFile() (string, error) {
	globalPath, err := GetGlobalPathConfig()
	if err != nil {
		return "", err
	}
	return filepath.Join(globalPath, authJsonFileName), nil
}

func ConfigJsonFile() (string, error) {
	globalPath, err := GetGlobalPathConfig()
	if err != nil {
		return "", err
	}
	return filepath.Join(globalPath, "config.json"), nil
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func GetGlobalPathConfig() (string, error) {
	dirname := "com.vercel.cli"

	bases := append([]string{}, xdg.ConfigDirs...)
	bases = append(bases, xdg.ConfigHome)
	bases = append(bases, xdg.DataDirs...)
	bases = append(bases, xdg.DataHome)

	// vercel CLI (xdg-app-paths) stores config.json/auth.json under a "Data"
	// subdir on Windows, so probe that before the bare app dir.
	var candidates []string
	for _, base := range bases {
		root := filepath.Join(base, dirname)
		candidates = append(candidates, filepath.Join(root, "Data"), root)
	}

	// prefer a dir that actually holds the vercel config/auth files
	for _, c := range candidates {
		if fileExists(filepath.Join(c, "config.json")) || fileExists(filepath.Join(c, authJsonFileName)) {
			return c, nil
		}
	}

	// fallback: first existing directory
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}

	return "", errors.New("global path config not found")
}
