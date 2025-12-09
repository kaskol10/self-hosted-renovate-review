# Self-Hosted Renovate Review

[![GitHub Actions](https://img.shields.io/badge/GitHub%20Actions-Marketplace-blue)](https://github.com/marketplace/actions/self-hosted-renovate-review)
[![Release](https://img.shields.io/github/v/release/kaskol10/self-hosted-renovate-review)](https://github.com/kaskol10/self-hosted-renovate-review/releases)

An AI-powered GitHub Action that automatically analyzes Renovate bot PRs for breaking changes and provides detailed impact assessments before merging dependency updates. Uses self-hosted LLM models (vLLM/LiteLLM) with Self Hosted models - no external API costs.

![](screenshot.png)

## Usage 

```yaml
name: Analyze Renovate PR

on:
  pull_request:
    types: [opened, synchronize, reopened]
    paths:
    - 'package.json'
    - 'package-lock.json'
    - 'yarn.lock'
    - 'pnpm-lock.yaml'
    - 'requirements.txt'
    - 'Pipfile'
    - 'poetry.lock'
    - 'go.mod'
    - 'go.sum'
    - 'Cargo.toml'
    - 'Cargo.lock'
    - 'pom.xml'
    - 'build.gradle'
    - '*.csproj'
    - '*.sln'
    - '**/*.yml'
    - '**/*.yaml'
    - '**/docker-compose*.yml'
    - '**/docker-compose*.yaml'
    - '**/Dockerfile*'
jobs:
  analyze:
    # Only run on PRs from Renovate bot
    if: github.event.pull_request.user.login == 'renovate[bot]' || github.event.pull_request.user.login == 'renovate'
    runs-on: self-hosted

    permissions:
      contents: read
      pull-requests: write  # for commenting on the PR 
    
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          ref: ${{ github.event.pull_request.head.ref }}
      
      - name: Analyze PR
        uses: kaskol10/self-hosted-renovate-review@main
        with:
          llm_api_url: "https://vllm.xxx.com"
          llm_api_key: 'not-needed'
          llm_provider: 'vllm'
          llm_model: "Qwen/Qwen2.5-7B-Instruct"
```


## 📥 Inputs

### Required Inputs

| Input | Description |
|-------|-------------|
| `llm_api_url` | LLM API URL. For self-hosted runners use `http://localhost:8000/v1`, for remote servers use `https://your-server.com/v1`. Automatically normalized if `/chat/completions` is included. |
| `llm_api_key` | LLM API key. Use `"not-needed"` if authentication is not required (e.g., vLLM on self-hosted runners). |
| `llm_provider` | LLM provider: `"vllm"` for direct vLLM server or `"litellm"` for LiteLLM proxy. |
| `llm_model` | LLM model name (e.g., `Qwen/Qwen2.5-7B-Instruct`). |

### Optional Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `github_token` | `$GITHUB_TOKEN` | GitHub token for API access. Automatically uses `$GITHUB_TOKEN` environment variable if not provided. |
| `repo` | `${{ github.repository }}` | Repository name in format `owner/repo`. Automatically uses `github.repository` from workflow context. |
| `pr_number` | `${{ github.event.pull_request.number }}` | Pull request number. Automatically uses `github.event.pull_request.number` from workflow context. |
| `go_version` | `'1.21'` | Go version to use for building the analyzer. |
| `checkout_path` | `'self-hosted-renovate-review'` | Path where the action repository will be checked out. |
| `action_repo` | `''` | Repository to checkout for the analyzer. Defaults to the action repository automatically. |
| `action_ref` | `''` | Git ref (branch, tag, or commit SHA) to checkout for building the analyzer. Optional: if using a specific version tag in the `uses` line (e.g., `@v0.1.0`), you can specify the same ref here (e.g., `v0.1.0`) to ensure the built code matches the action version. Leave empty to use the default branch. |

## 📤 Outputs

The action posts a detailed analysis comment directly on the Pull Request containing:

- ✅ **Breaking Changes Risk Assessment**
- 📊 **Impact Analysis**
- 🔄 **Migration Requirements**
- 🧪 **Testing Recommendations**
- 🎯 **Confidence Level**
- 💡 **Recommendation**

The comment is automatically posted after analysis completes. If analysis fails, the action exits with an error code.

## 🎯 Features

- **Automatic Analysis**: Triggers automatically on Renovate bot PRs
- **Breaking Change Detection**: Identifies potential breaking changes based on semantic versioning
- **Impact Assessment**: Analyzes which parts of your codebase might be affected
- **Migration Guidance**: Provides actionable recommendations for handling updates
- **Testing Recommendations**: Suggests what should be tested
- **PR Comments**: Posts detailed analysis directly on the PR
- **Self-Hosted AI**: Uses vLLM/LiteLLM with self hosted models (no external API costs)

## 🚀 Quick Start

### 1. Prerequisites

- A GitHub repository with Renovate bot configured
- A vLLM or LiteLLM server **reachable from your GitHub Actions runners** (see [LLM Setup](#2-llm-setup))
  - For GitHub-hosted runners: The server must be accessible via a public URL or VPN
  - For self-hosted runners: The server can run on `localhost` or be accessible via network

### 2. LLM Setup

> **Important**: Your vLLM or LiteLLM server must be **reachable from your GitHub Actions runners**. 
> - **Self-hosted runners**: The server can run on `localhost` on the same machine
> - **GitHub-hosted runners**: The server must be accessible via a public URL or through a VPN/tunnel

#### Option A: Direct vLLM

```bash
# Install vLLM
pip install vllm

# Start vLLM server
python -m vllm.entrypoints.openai.api_server \
    --model Qwen/Qwen2.5-7B-Instruct \
    --port 8000 \
    --trust-remote-code
```

**For self-hosted runners**: Run the server on `localhost:8000` on the same machine as the runner.

**For GitHub-hosted runners**: Deploy the server on a machine accessible via network (public URL or VPN) and use the full URL in `LLM_API_URL`.

#### Option B: LiteLLM Proxy (Recommended for Production)

LiteLLM provides caching, load balancing, and monitoring:

```bash
# Install LiteLLM
pip install litellm

# Start LiteLLM (connects to vLLM)
litellm --port 4000 --config config.yaml
```

**For self-hosted runners**: Run LiteLLM on `localhost:4000` on the same machine.

**For GitHub-hosted runners**: Deploy LiteLLM on a machine accessible via network and use the full URL in `LLM_API_URL`.

### 3. Configure GitHub Secrets

Go to **Settings** → **Secrets and variables** → **Actions** and add:

- **`LLM_API_URL`**: Your LLM API URL (must be reachable from your GitHub Actions runners)
  - **Self-hosted runners**: `http://localhost:8000/v1` (vLLM) or `http://localhost:4000/v1` (LiteLLM)
  - **GitHub-hosted runners**: `https://your-server.com/v1` (must be publicly accessible or via VPN)
- **`LLM_API_KEY`** (optional): API key if required, or set to `"not-needed"` for vLLM without auth
- **`LLM_PROVIDER`** (optional): `"vllm"` (default) or `"litellm"`
- **`LLM_MODEL`** (optional): Model name (e.g., `Qwen/Qwen2.5-7B-Instruct`)

> **Note**: The action automatically normalizes URLs ending with `/chat/completions` to `/v1`.
> 
> **Important**: Ensure your LLM server is reachable from your runners. For GitHub-hosted runners, you may need to:
> - Expose the server via a public URL (with proper security/authentication)
> - Use a VPN or tunnel service to make the server accessible
> - Consider using a self-hosted runner if you want to use `localhost`

### 4. Create Workflow

Create `.github/workflows/analyze-renovate-pr.yml`:

```yaml
name: Analyze Renovate PR

on:
  pull_request:
    types: [opened, synchronize, reopened]
    paths:
      - 'package.json'
      - 'package-lock.json'
      - 'go.mod'
      - 'go.sum'
      - 'requirements.txt'
      - '**/*.yml'
      - '**/*.yaml'

jobs:
  analyze:
    # Only run on PRs from Renovate bot
    if: github.event.pull_request.user.login == 'renovate[bot]' || github.event.pull_request.user.login == 'renovate'
    runs-on: ubuntu-latest  # Use 'self-hosted' for self-hosted runners
    permissions:
      contents: read
      pull-requests: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          ref: ${{ github.event.pull_request.head.ref }}

      - name: Analyze PR
        uses: kaskol10/self-hosted-renovate-review@v0.1.0
        with:
          llm_api_url: ${{ secrets.LLM_API_URL }}
          llm_api_key: ${{ secrets.LLM_API_KEY || 'not-needed' }}
          llm_provider: ${{ secrets.LLM_PROVIDER || 'vllm' }}
          llm_model: ${{ secrets.LLM_MODEL || 'Qwen/Qwen2.5-7B-Instruct' }}
```

**That's it!** The action automatically uses:
- `github.repository` for the repo name
- `github.event.pull_request.number` for the PR number
- `github.token` for authentication (automatically provided)

> **Note**: The workflow needs `permissions` set to allow the action to read PRs and write comments. See the example above.


## 📖 Usage Examples

### Basic Usage (GitHub-Hosted Runner)

```yaml
- name: Analyze PR
  uses: kaskol10/self-hosted-renovate-review@v0.1.0
  with:
    llm_api_url: ${{ secrets.LLM_API_URL }}
    llm_api_key: 'not-needed'
    llm_provider: 'vllm'
    llm_model: 'Qwen/Qwen2.5-7B-Instruct'
```

### Self-Hosted Runner

For self-hosted runners with vLLM running locally:

```yaml
jobs:
  analyze:
    runs-on: self-hosted
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Analyze PR
        uses: kaskol10/self-hosted-renovate-review@main
        with:
          llm_api_url: 'http://localhost:8000/v1'
          llm_api_key: 'not-needed'
          llm_provider: 'vllm'
          llm_model: 'Qwen/Qwen2.5-7B-Instruct'
```

### Advanced Configuration

```yaml
      - name: Analyze PR
        uses: kaskol10/self-hosted-renovate-review@v0.1.0
  with:
    llm_api_url: ${{ secrets.LLM_API_URL }}
    llm_api_key: ${{ secrets.LLM_API_KEY || 'not-needed' }}
    llm_provider: 'litellm'
    llm_model: 'Qwen/Qwen2.5-14B-Instruct'
    # Optional overrides (usually not needed)
    github_token: ${{ secrets.GITHUB_TOKEN }}
    repo: ${{ github.repository }}
    pr_number: ${{ github.event.pull_request.number }}
    go_version: '1.22'
```


## 📋 How It Works

1. **Trigger**: When Renovate bot opens or updates a PR, the workflow triggers
2. **Build**: The Go analyzer is compiled in the GitHub Actions environment
3. **Analysis**: The analyzer:
   - Fetches PR details and changed files from GitHub API
   - Extracts dependency version changes from diffs
   - Sends analysis request to your LLM server
   - Generates a comprehensive impact report
4. **Comment**: Posts the analysis as a comment on the PR

## 🔧 Supported Dependency Files

The analyzer automatically detects changes in:

- **Package Managers**: `package.json`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `requirements.txt`, `Pipfile`, `poetry.lock`, `go.mod`, `go.sum`, `Cargo.toml`, `Cargo.lock`, `pom.xml`, `build.gradle`, `*.csproj`, `*.sln`
- **Docker Images**: YAML files (Kubernetes, docker-compose, CI/CD) with `image:` fields

## 🛠️ Troubleshooting

### Workflow not triggering

- Ensure the workflow file is in `.github/workflows/`
- Check that Renovate bot's username matches (usually `renovate[bot]`)
- Verify the PR modifies dependency files listed in `paths`

### LLM connection errors

- Verify your LLM server is running and reachable from your GitHub Actions runner:
  - Self-hosted runners: `curl http://localhost:8000/v1/models` (from the runner machine)
  - GitHub-hosted runners: `curl https://your-server.com/v1/models` (from a public network)
- Check the `LLM_API_URL` secret is correct and accessible from the runner's network
- For GitHub-hosted runners, ensure the server is publicly accessible or accessible via VPN
- Review workflow logs for detailed error messages
- If using `localhost`, ensure you're using a self-hosted runner (GitHub-hosted runners cannot access `localhost`)

### Build errors

- Ensure Go 1.21+ is available (or specify `go_version` input)
- Check that all dependencies are in `go.mod`

### Permission errors (403 Resource not accessible by integration)

If you see `403 Resource not accessible by integration`, the workflow needs explicit permissions:

```yaml
jobs:
  analyze:
    permissions:
      contents: read
      pull-requests: write
    # ... rest of job
```

The action needs:
- `contents: read` - to read repository contents and PR data
- `pull-requests: write` - to post comments on pull requests

## 🔒 Security

- The workflow only runs on PRs from Renovate bot
- API keys are stored as GitHub Secrets
- The analyzer only reads PR data, never modifies code
- LLM runs on your infrastructure (data stays private)

## 🧪 Local Development

For local testing:

```bash
# Build the analyzer
go build -o analyzer ./cmd/analyzer

# Run locally
./analyzer \
  --repo "owner/repo" \
  --pr-number 123 \
  --github-token "your-token" \
  --llm-provider vllm \
  --llm-url "http://localhost:8000/v1" \
  --llm-key "" \
  --llm-provider vllm
```


## 📦 Releases

See [CHANGELOG.md](./CHANGELOG.md) for a detailed list of changes in each version.

**Latest Release**: [v0.1.3](https://github.com/kaskol10/self-hosted-renovate-review/releases/tag/v0.1.3)

**All Releases**: [View on GitHub](https://github.com/kaskol10/self-hosted-renovate-review/releases)

### Versioning

- **`@v0.1.3`** - Use a specific version (recommended for production)
- **`@v0.1`** - Use the latest v0.1.x version
- **`@main`** - Use the latest development version (may have breaking changes)

## 📝 License

[MIT](LICENSE)

## 🤝 Contributing

Contributions welcome! Feel free to open issues or PRs.
