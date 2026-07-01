# crd-getschema

Downloads JSON schemas for Custom Resource Definitions (CRDs) installed in a Kubernetes cluster. The schemas are compatible with validation tools like [kubeconform](https://github.com/yannh/kubeconform) and IDE autocompletion plugins.

## Output structure

Schemas are written under a `GROUP/VERSION/` hierarchy:

```
schemas/
  cilium.io/
    v2alpha1/
      CiliumBGPClusterConfig.json
  monitoring.coreos.com/
    v1/
      Prometheus.json
      ServiceMonitor.json
```

## Installation

```bash
go install github.com/netops2devops/crdschema@latest
```

Or build from source:

```bash
git clone <repo-url>
cd crd-getschema
go build -o getschema .
```

## Usage

```
getschema [flags]

Flags:
      --all                 Download schemas for all CRDs (overrides --crd when both are set)
      --crd string          Download schema for a specific CRD (full name like 'widgets.example.com' or kind like 'Widget')
  -h, --help                help for getschema
      --kubeconfig string   Path to kubeconfig file (default "$HOME/.kube/config")
  -o, --output-dir string   Directory where schemas will be stored (default "./schemas")
```

### Examples

Download schemas for every CRD in the cluster:

```bash
getschema --all
```

Download to a custom directory:

```bash
getschema --all --output-dir /tmp/schemas
```

Download the schema for a single CRD (accepts the full resource name or just the kind):

```bash
getschema --crd ciliumbgpclusterconfigs.cilium.io
getschema --crd CiliumBGPClusterConfig
```

Use a non-default kubeconfig:

```bash
getschema --all --kubeconfig ~/.kube/staging.yaml
```

## Using schemas with kubeconform

Point kubeconform at the output directory using its `{{ .Group }}/{{ .NormalizedVersion }}/{{ .ResourceKind }}` location template:

```bash
kubeconform \
  -schema-location default \
  -schema-location './schemas/{{ .Group }}/{{ .NormalizedVersion }}/{{ .ResourceKind }}.json' \
  manifests/
```

## Requirements

- Go 1.22+
- A reachable Kubernetes cluster with a valid kubeconfig
