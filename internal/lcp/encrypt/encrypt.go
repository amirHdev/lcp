package encrypt

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/amirhdev/ebook-lcp-server/internal/config"
)

// Encrypter defines the behavior required by the publication use case.
type Encrypter interface {
	Encrypt(inputPath, contentID, filename string) (string, error)
}

// ReadiumEncrypter wraps the upstream lcpencrypt CLI so the repository
// produces real LCP-protected files and not local copies.
type ReadiumEncrypter struct {
	cfg *config.Config
}

func NewReadiumEncrypter(cfg *config.Config) *ReadiumEncrypter {
	return &ReadiumEncrypter{cfg: cfg}
}

func (e *ReadiumEncrypter) Encrypt(inputPath, contentID, filename string) (string, error) {
	if e.cfg == nil {
		return "", fmt.Errorf("missing LCP configuration")
	}

	outputDir := e.cfg.LCP.Storage.FS.Directory
	if outputDir == "" {
		outputDir = filepath.Join(filepath.Dir(inputPath), "storage")
	}

	providerURI := strings.TrimSpace(e.cfg.LCP.ProviderURI)
	if providerURI == "" {
		providerURI = strings.TrimSpace(e.cfg.Server.PublicBaseURL)
	}
	if providerURI == "" {
		return "", fmt.Errorf("missing provider uri")
	}
	storageURL := strings.TrimSpace(e.cfg.Server.PublicBaseURL)
	if storageURL == "" {
		storageURL = providerURI
	}

	lcpCoreURL := strings.TrimSpace(e.cfg.LCP.CoreURL)
	if lcpCoreURL == "" {
		return "", fmt.Errorf("missing LCP core url")
	}

	args := []string{
		"-input", inputPath,
		"-provider", providerURI,
		"-storage", outputDir,
		"-url", strings.TrimRight(storageURL, "/") + "/publications",
		"-temp", filepath.Dir(inputPath),
		"-contentid", contentID,
		"-lcpsv", lcpCoreURL,
	}
	if e.cfg.LCP.CoreUser != "" {
		args = append(args, "-login", e.cfg.LCP.CoreUser)
	}
	if e.cfg.LCP.CorePass != "" {
		args = append(args, "-password", e.cfg.LCP.CorePass)
	}
	cmd := exec.CommandContext(context.Background(), "/usr/local/bin/lcpencrypt", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if recovered, ok := recoverEncryptedPath(outputDir, contentID, inputPath, string(output)); ok {
			return recovered, nil
		}
		return "", fmt.Errorf("lcpencrypt failed: %w: %s", err, string(output))
	}

	expected := filepath.Join(outputDir, contentID+filepath.Ext(inputPath))
	if filepath.Ext(inputPath) == ".pdf" {
		expected = filepath.Join(outputDir, contentID+".lcpdf")
	}
	if filepath.Ext(inputPath) == ".epub" {
		expected = filepath.Join(outputDir, contentID+".epub")
	}
	return expected, nil
}

func recoverEncryptedPath(outputDir, contentID, inputPath, output string) (string, bool) {
	if !strings.Contains(output, "Error notifying the LCP Server") && !strings.Contains(output, "connect: connection refused") {
		return "", false
	}
	dirs := []string{
		outputDir,
		"/var/lib/lcp/storage",
		filepath.Join(filepath.Dir(inputPath), "storage"),
	}
	candidates := []string{}
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		candidates = append(
			candidates,
			filepath.Join(dir, contentID+filepath.Ext(inputPath)),
			filepath.Join(dir, contentID+".lcpdf"),
			filepath.Join(dir, contentID+".epub"),
			filepath.Join(dir, contentID+".lcp"),
		)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}
