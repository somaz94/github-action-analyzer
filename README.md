# GitHub Action Analyzer

[![License](https://img.shields.io/github/license/somaz94/github-action-analyzer)](https://github.com/somaz94/github-action-analyzer)
![Latest Tag](https://img.shields.io/github/v/tag/somaz94/github-action-analyzer)
![Top Language](https://img.shields.io/github/languages/top/somaz94/github-action-analyzer?color=green&logo=go&logoColor=b)
[![GitHub Marketplace](https://img.shields.io/badge/Marketplace-GitHub%20Action%20Analyzer-blue?logo=github)](https://github.com/marketplace/actions/github-action-analyzer)

## Overview

The **GitHub Action Analyzer** is a GitHub Action that analyzes your workflow performance and provides optimization recommendations. It helps identify bottlenecks, suggests caching strategies, and offers Docker optimization tips to improve your CI/CD pipeline efficiency.

<br/>

## Inputs

| Input           | Required | Description                                    | Default | Example                |
|----------------|----------|------------------------------------------------|---------|------------------------|
| `github_token` | Yes      | GitHub token for API access                    | -       | `${{ secrets.GITHUB_TOKEN }}` |
| `workflow_file`| Yes      | Name of the workflow file to analyze          | -       | `"ci.yml"`            |
| `repository`   | Yes      | Repository in owner/repo format               | -       | `"owner/repo"`        |

<br/>

## Example Usage

```yaml
name: Analyze Workflow
on: [push, workflow_dispatch]

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Analyze Workflow Performance
        uses: somaz94/github-action-analyzer@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          workflow_file: ci.yml
          repository: ${{ github.repository }}
```

<br/>

## Features

- Workflow runtime analysis
- Cache hit rate monitoring
- Docker layer optimization suggestions
- Job dependency analysis
- Resource usage tracking
- Step duration analysis
- Automated recommendations for:
  - Caching strategies
  - Docker image optimization
  - Workflow structure improvements
  - Resource allocation

## Supported Languages & Frameworks

| Language/Framework | Cache Recommendations | Version Check |
|-------------------|----------------------|---------------|
| Go                | ✅                   | ✅            |
| Node.js          | ✅                   | ✅            |
| Python           | ✅                   | ✅            |
<!-- | Java/Maven       | ✅                   | ✅            |
| Ruby             | ✅                   | ✅            |
| Rust             | ✅                   | ✅            |
| .NET             | ✅                   | ✅            | -->

Each language includes specific recommendations for:
- Dependencies caching
- Build artifacts caching
- Version updates
- Best practices for the ecosystem

<br/>

## Advanced Usage

### Analyzing Specific Workflows

```yaml
- uses: somaz94/github-action-analyzer@v1
  with:
    github_token: ${{ secrets.GITHUB_TOKEN }}
    workflow_file: deploy.yml
    repository: ${{ github.repository }}
```

### Monitoring Multiple Workflows

```yaml
- name: Analyze CI Workflow
  uses: somaz94/github-action-analyzer@v1
  with:
    workflow_file: ci.yml
    
- name: Analyze Deploy Workflow
  uses: somaz94/github-action-analyzer@v1
  with:
    workflow_file: deploy.yml
```

<br/>

## Analysis Types

### 1. Performance Analysis
- Job execution time trends
- Step duration breakdown
- Resource utilization patterns
- Bottleneck identification

### 2. Cache Analysis
- Cache hit/miss ratios
- Cache size monitoring
- Cache restoration times
- Optimization suggestions

### 3. Docker Analysis
- Layer caching effectiveness
- Image size optimization
- Build time analysis
- Multi-stage build recommendations

<br/>

## Troubleshooting

Common issues and solutions:

1. **API Rate Limiting**
   - Error: `API rate limit exceeded`
   - Solution: Use a GitHub token with appropriate permissions

2. **Access Denied**
   - Error: `Resource not accessible`
   - Solution: Ensure the token has `repo` scope access

3. **Invalid Workflow File**
   - Error: `Workflow file not found`
   - Solution: Verify the workflow file path and name

<br/>

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

<br/>

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.