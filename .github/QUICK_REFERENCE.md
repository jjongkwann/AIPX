# CI/CD Quick Reference

## Workflow Overview

| Workflow | Trigger | Duration | Status |
|----------|---------|----------|--------|
| Proto Lint | `shared/proto/**` | ~5min | Required |
| Go Tests | `**/*.go` | ~10min | Required |
| Python Tests | `**/*.py` | ~15min | Required |
| Docker Build | `**/Dockerfile` | ~20min | Required |
| Integration Tests | All + Daily | ~30min | Optional |

## Common Commands

### Local Testing

```bash
# Proto
cd shared/proto && buf lint && buf format -d --exit-code

# Go
cd shared/go && go vet ./... && golangci-lint run && go test -v -race ./...

# Python
cd shared/python && ruff check aipx/ && black --check aipx/ && pytest tests/ -v

# Docker
docker-compose build && docker-compose up -d
```

### Fix Issues

```bash
# Auto-fix Python formatting
cd shared/python
ruff check aipx/ --fix
ruff format aipx/
black aipx/
isort aipx/

# Auto-fix Proto formatting
cd shared/proto
buf format -w

# Auto-fix Go imports
cd shared/go
goimports -w .
```

## Status Badges

```markdown
![Proto Lint](https://github.com/YOUR_USERNAME/AIPX/workflows/Proto%20Lint/badge.svg)
![Go Tests](https://github.com/YOUR_USERNAME/AIPX/workflows/Go%20Tests/badge.svg)
![Python Tests](https://github.com/YOUR_USERNAME/AIPX/workflows/Python%20Tests/badge.svg)
![Docker Build](https://github.com/YOUR_USERNAME/AIPX/workflows/Docker%20Build/badge.svg)
```

## Workflow Files

- `.github/workflows/proto-lint.yml` - Proto validation
- `.github/workflows/go-test.yml` - Go testing
- `.github/workflows/python-test.yml` - Python testing
- `.github/workflows/docker-build.yml` - Docker builds
- `.github/workflows/integration-test.yml` - Integration tests
- `.github/dependabot.yml` - Dependency updates
- `.golangci.yml` - Go linter config
- `shared/python/pyproject.toml` - Python tools config

## Debugging

```bash
# Test locally with act
brew install act
act -l  # List workflows
act -j lint  # Run specific job
act pull_request  # Simulate PR

# Check workflow syntax
yamllint .github/workflows/*.yml

# View logs
# GitHub repo > Actions tab > Select workflow > View logs
```

## Required Secrets

| Secret | Required | Purpose |
|--------|----------|---------|
| `CODECOV_TOKEN` | No | Coverage reporting |
| `GITHUB_TOKEN` | Auto | Package push (auto-provided) |

## Cache Keys

- Go: `${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}`
- Python: `${{ runner.os }}-pip-${{ hashFiles('**/requirements*.txt') }}`
- Docker: `${{ runner.os }}-buildx-${{ github.sha }}`

## Troubleshooting

### Workflow Not Running
- Check trigger paths in workflow file
- Verify branch protection rules
- Check Actions permissions in Settings

### Cache Issues
- Settings > Actions > Caches > Delete cache
- Update cache key in workflow

### GHCR Push Failed
- Settings > Actions > General
- Workflow permissions: Read and write
- Enable "Allow GitHub Actions to create and approve pull requests"

## Quick Links

- [Workflows README](.github/workflows/README.md)
- [Full Setup Summary](../CI_CD_SETUP_SUMMARY.md)
- [GitHub Actions Docs](https://docs.github.com/en/actions)
