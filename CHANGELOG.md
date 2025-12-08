# Changelog

All notable changes to the Self-Hosted Renovate Review action will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-12-08

### Added
- Initial release of Self-Hosted Renovate Review GitHub Action
- AI-powered analysis of Renovate bot PRs using self-hosted LLM models
- Support for vLLM and LiteLLM providers
- Automatic breaking change detection based on semantic versioning
- Impact assessment for dependency updates
- Migration guidance and testing recommendations
- Support for multiple package managers:
  - Node.js (package.json, package-lock.json, yarn.lock, pnpm-lock.yaml)
  - Python (requirements.txt, Pipfile, poetry.lock)
  - Go (go.mod, go.sum)
  - Rust (Cargo.toml, Cargo.lock)
  - Java (pom.xml, build.gradle)
  - .NET (*.csproj, *.sln)
- Docker image change detection in YAML files
- Automatic PR comment posting with detailed analysis
- Support for both GitHub-hosted and self-hosted runners
- Configurable Go version for building the analyzer
- Automatic GitHub context variable detection (repository, PR number, token)
- LLM URL normalization (handles /chat/completions and /v1 suffixes)
- Support for release tags and version pinning via `action_ref` input

### Features
- **Breaking Change Detection**: Identifies potential breaking changes based on semantic versioning
- **Impact Analysis**: Analyzes which parts of the codebase might be affected
- **Migration Guidance**: Provides actionable recommendations for handling updates
- **Testing Recommendations**: Suggests what should be tested
- **Self-Hosted AI**: Uses vLLM/LiteLLM with QWEN models (no external API costs)
- **Flexible Configuration**: Supports both direct vLLM and LiteLLM proxy setups

