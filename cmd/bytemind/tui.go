package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bytemind/internal/assets"
	"bytemind/internal/config"
	"bytemind/internal/tui"
)

var runTUIProgram = tui.Run

func runTUI(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("config", "", "Path to config file")
	model := fs.String("model", "", "Override model name")
	sessionID := fs.String("session", "", "Resume an existing session")
	streamOverride := fs.String("stream", "", "Override streaming: true or false")
	workspaceOverride := fs.String("workspace", "", "Workspace to operate on; defaults to current directory")
	maxIterations := fs.Int("max-iterations", 0, "Override execution budget for this run")

	if err := fs.Parse(args); err != nil {
		return err
	}

	workspace, err := resolveWorkspace(*workspaceOverride)
	if err != nil {
		return err
	}
	if err := ensureAPIConfigForTUI(workspace, *configPath, stdin, stdout); err != nil {
		return err
	}

	app, store, sess, err := bootstrap(*configPath, *model, *sessionID, *streamOverride, *workspaceOverride, *maxIterations, stdin, stdout)
	if err != nil {
		return err
	}

	cfg, err := config.Load(workspace, *configPath)
	if err != nil {
		return err
	}
	if *model != "" {
		cfg.Provider.Model = *model
	}
	if *streamOverride != "" {
		parsed, err := strconv.ParseBool(*streamOverride)
		if err != nil {
			return err
		}
		cfg.Stream = parsed
	}
	if *maxIterations > 0 {
		cfg.MaxIterations = *maxIterations
	}
	home, err := config.EnsureHomeLayout()
	if err != nil {
		return err
	}
	imageStore, err := assets.NewFileAssetStore(home)
	if err != nil {
		return err
	}

	return runTUIProgram(tui.Options{
		Runner:     app,
		Store:      store,
		Session:    sess,
		ImageStore: imageStore,
		Config:     cfg,
		Workspace:  sess.Workspace,
	})
}

func ensureAPIConfigForTUI(workspace, configPath string, stdin io.Reader, stdout io.Writer) error {
	cfg, err := config.Load(workspace, configPath)
	if err != nil {
		if strings.TrimSpace(configPath) != "" && errors.Is(err, os.ErrNotExist) {
			return runInteractiveConfigSetup(workspace, configPath, config.Default(workspace), stdin, stdout)
		}
		return err
	}
	if strings.TrimSpace(cfg.Provider.ResolveAPIKey()) != "" {
		return nil
	}
	return runInteractiveConfigSetup(workspace, configPath, cfg, stdin, stdout)
}

func runInteractiveConfigSetup(workspace, configPath string, cfg config.Config, stdin io.Reader, stdout io.Writer) error {
	reader := bufio.NewReader(stdin)
	fmt.Fprintln(stdout, "未检测到可用 API 配置，请先完成初始化。")
	fmt.Fprintln(stdout, "配置格式：OpenAI-compatible（依次输入 url / key / model）。")

	baseURL, err := promptSetupValue(reader, stdout, "url")
	if err != nil {
		return err
	}
	apiKey, err := promptSetupValue(reader, stdout, "key")
	if err != nil {
		return err
	}
	modelName, err := promptSetupValue(reader, stdout, "model")
	if err != nil {
		return err
	}

	baseURL, err = validateBaseURL(baseURL)
	if err != nil {
		return err
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return errors.New("配置失败: API Key 不能为空")
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return errors.New("配置失败: Model 不能为空")
	}

	cfg.Provider.Type = "openai-compatible"
	cfg.Provider.AutoDetectType = false
	cfg.Provider.BaseURL = baseURL
	cfg.Provider.APIPath = ""
	cfg.Provider.Model = modelName
	cfg.Provider.APIKey = apiKey
	cfg.Provider.APIKeyEnv = ""
	cfg.Provider.AuthHeader = ""
	cfg.Provider.AuthScheme = ""
	cfg.Provider.ExtraHeaders = nil

	targetPath, err := resolveSetupConfigPath(workspace, configPath)
	if err != nil {
		return err
	}
	if err := writeConfigFile(targetPath, cfg); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "配置已写入: %s\n", targetPath)
	return nil
}

func promptSetupValue(reader *bufio.Reader, stdout io.Writer, label string) (string, error) {
	fmt.Fprintf(stdout, "%s: ", strings.TrimSpace(label))

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		if errors.Is(err, io.EOF) {
			return "", errors.New("初始化已取消: 未收到输入")
		}
		return "", fmt.Errorf("配置失败: %s 不能为空", label)
	}
	return line, nil
}

func validateBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("配置失败: Base URL 不能为空")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("配置失败: Base URL 必须是合法的 http(s) 地址")
	}
	return strings.TrimRight(value, "/"), nil
}

func resolveSetupConfigPath(workspace, configPath string) (string, error) {
	if strings.TrimSpace(configPath) != "" {
		return filepath.Abs(configPath)
	}
	return filepath.Join(workspace, "config.json"), nil
}

func writeConfigFile(path string, cfg config.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
