---
icon: lucide/workflow
---

# CI/CD Pipeline

Patrizio uses GitHub Actions for continuous integration and deployment.

## Workflow files

| Step     | Description                                                                                     |
|----------|-------------------------------------------------------------------------------------------------|
| `Commit` | Runs `markdownlint` and `golangci-lint`. Ensures tests pass.                                    |
| `Docker` | Builds the docker image and pushes it in the GitHub Container Registry (GHCR) of the project    |
| `Docs`   | Publishes the project website upon a new commit landing in `main` or when a new release happens |

For more details about the implementations, see the `.github/workflows/` folder in the repository.

## Pre-commit

An additional tool that is used by the project is `pre-commit`. It allows to enforce some standardization across the
codebase, especially regarding linting and formatting. It also enforces test run at push time. This is particularly
useful when pushing directly into the `main` branch. Also, when using AI assistance, `pre-commit` acts as guardrail
ensuring the codebase written already meets a lower bar before human review. Finally, tests upon committing force the AI
to actually have a look at them before pushing.

!!! note
    Note that AI can still get around these checks. Nothing stops them from using the good old `git push --no-verify`.

## The logic behind these steps

The idea of CI/CD steps together with Pre-commit checks is to ensure that the project stays stable. Finally, further
automation is planned for automatic changelogs and releases.
