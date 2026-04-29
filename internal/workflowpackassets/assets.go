package workflowpackassets

import "embed"

const BuiltInManifestPath = "builtins/v0/manifest.json"

//go:embed builtins/v0/*.json
var builtInWorkflowPackFS embed.FS

func BuiltInFS() embed.FS {
	return builtInWorkflowPackFS
}
