package projectsubstrate

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

type runeContextMutationProvider interface {
	RenderConfig(version, sourceType, sourcePath string) (string, error)
	RenderAssuranceBaseline(sourceType, adoptionCommit string) (string, error)
}

const zeroAdoptionCommit = "0000000000000000000000000000000000000000"

type canonicalInitializationMutation struct {
	ConfigYAML           string
	AssuranceBaselineYML string
	SourcePath           string
	AssurancePath        string
	BaselinePath         string
}

var canonicalProvider runeContextMutationProvider = newBundledRuneContextMutationProvider()

func canonicalInitialization(version, sourceType string) (canonicalInitializationMutation, error) {
	config, err := canonicalProvider.RenderConfig(version, sourceType, CanonicalSourcePath)
	if err != nil {
		return canonicalInitializationMutation{}, err
	}
	baseline, err := canonicalProvider.RenderAssuranceBaseline(sourceType, zeroAdoptionCommit)
	if err != nil {
		return canonicalInitializationMutation{}, err
	}
	return canonicalInitializationMutation{
		ConfigYAML:           config,
		AssuranceBaselineYML: baseline,
		SourcePath:           CanonicalSourcePath,
		AssurancePath:        CanonicalAssurancePath,
		BaselinePath:         canonicalAssuranceBaselinePath,
	}, nil
}

func canonicalUpgradeConfig(version, sourceType string) (string, error) {
	return canonicalProvider.RenderConfig(version, sourceType, CanonicalSourcePath)
}

//go:embed bundled_runecontext/v0/*.tmpl bundled_runecontext/v0/assurance/*.tmpl
var bundledRuneContextTemplatesFS embed.FS

type bundledRuneContextMutationProvider struct {
	templates *template.Template
}

func newBundledRuneContextMutationProvider() bundledRuneContextMutationProvider {
	templates := template.Must(template.New("bundled_runecontext").ParseFS(
		bundledRuneContextTemplatesFS,
		"bundled_runecontext/v0/*.tmpl",
		"bundled_runecontext/v0/assurance/*.tmpl",
	))
	return bundledRuneContextMutationProvider{templates: templates}
}

func (p bundledRuneContextMutationProvider) RenderConfig(version, sourceType, sourcePath string) (string, error) {
	payload := map[string]string{
		"RuneContextVersion": yamlScalar(defaultRuneContextVersion(version)),
		"SourceType":         yamlScalar(defaultSourceType(sourceType)),
		"SourcePath":         yamlScalar(defaultSourcePath(sourcePath)),
	}
	out, err := p.executeTemplate("runecontext.yaml.tmpl", payload)
	if err != nil {
		return "", fmt.Errorf("render canonical runecontext.yaml: %w", err)
	}
	return out, nil
}

func (p bundledRuneContextMutationProvider) RenderAssuranceBaseline(sourceType, adoptionCommit string) (string, error) {
	payload := map[string]string{
		"SourceType":     defaultSourceType(sourceType),
		"AdoptionCommit": yamlScalar(defaultAdoptionCommit(adoptionCommit)),
	}
	out, err := p.executeTemplate("baseline.yaml.tmpl", payload)
	if err != nil {
		return "", fmt.Errorf("render canonical assurance baseline: %w", err)
	}
	return out, nil
}

func defaultAdoptionCommit(adoptionCommit string) string {
	value := strings.TrimSpace(adoptionCommit)
	if value == "" {
		return zeroAdoptionCommit
	}
	return value
}

func (p bundledRuneContextMutationProvider) executeTemplate(path string, payload map[string]string) (string, error) {
	if p.templates == nil {
		return "", fmt.Errorf("template bundle unavailable")
	}
	var out bytes.Buffer
	if err := p.templates.ExecuteTemplate(&out, path, payload); err != nil {
		return "", err
	}
	if !strings.HasSuffix(out.String(), "\n") {
		out.WriteString("\n")
	}
	return out.String(), nil
}

func defaultRuneContextVersion(version string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		return recommendedRuneContextVersionTarget()
	}
	return v
}

func defaultSourceType(sourceType string) string {
	t := strings.TrimSpace(sourceType)
	if t == "" {
		return "embedded"
	}
	return t
}

func defaultSourcePath(sourcePath string) string {
	p := strings.TrimSpace(sourcePath)
	if p == "" {
		return CanonicalSourcePath
	}
	return p
}
