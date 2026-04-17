package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

type EntrypointRequest struct {
	WorkspaceOverride     string
	ConfigPath            string
	ModelOverride         string
	SessionID             string
	StreamOverride        string
	MaxIterationsOverride int
	RequireAPIKey         bool
	Stdin                 io.Reader
	Stdout                io.Writer
}

func BootstrapEntrypoint(req EntrypointRequest) (Runtime, error) {
	workspaceOverride := strings.TrimSpace(req.WorkspaceOverride)
	if workspaceOverride == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Runtime{}, err
		}
		workspaceOverride, err = filepath.Abs(cwd)
		if err != nil {
			return Runtime{}, err
		}
	}
	workspace, err := ResolveWorkspace(workspaceOverride)
	if err != nil {
		return Runtime{}, err
	}

	return Bootstrap(BootstrapRequest{
		Workspace:             workspace,
		ConfigPath:            req.ConfigPath,
		ModelOverride:         req.ModelOverride,
		SessionID:             req.SessionID,
		StreamOverride:        req.StreamOverride,
		MaxIterationsOverride: req.MaxIterationsOverride,
		RequireAPIKey:         req.RequireAPIKey,
		Stdin:                 req.Stdin,
		Stdout:                req.Stdout,
	})
}
