# Webtester CI/CD Workflow Requirements and Specification

## 1. Purpose

This document defines the requirements and specifications for the Webtester GitHub Actions workflows. The workflows separate automated Go testing from manually triggered Docker image builds.

The solution consists of two independent workflows:

1. **Test Webtester** — validates the code through Go tests and static analysis (golangci-lint).
2. **Build Webtester** — builds a versioned Docker image on the self-hosted runner installed on `oong-serv`.

## 2. Scope

### In scope

- Automated testing and linting on pushes and pull requests.
- Manual Docker image builds with a user-provided version.
- Native GitHub Actions branch selection when manually starting a build.
- Automatic repository checkout into the runner workspace.
- Local Docker image creation on the self-hosted runner.
- Basic input validation and build summary output.

### Out of scope

- Deploying or restarting the Webtester application/container.
- Pushing images to Docker Hub, GitHub Container Registry (GHCR), or another registry.
- Automatically requiring the test workflow to pass before a build workflow can run.
- Managing server SSH credentials or connecting to a different deployment server.
- Database migrations, seeders, application health checks, rollback, and release approval.

## 3. Environment and Assumptions

- The Webtester source code is hosted in GitHub.
- The repository contains a valid `Dockerfile` at its root.
- The repository contains Go modules and supports `go test ./...`.
- The Go version required by the project is `1.26`.
- A GitHub Actions self-hosted runner is installed and online on `oong-serv`.
- The self-hosted runner uses Linux x86-64 and has the labels `self-hosted`, `Linux`, and `X64`.
- The runner's operating-system user has permission to run Docker commands.
- Docker is installed and running on the self-hosted runner host.
- GitHub-hosted `ubuntu-latest` runners are used for the test workflow.

## 4. Workflow A: Test Webtester

### 4.1 Workflow file

`.github/workflows/test.yml`

### 4.2 Trigger conditions

The workflow runs on:

- `push` events.
- `pull_request` events.

No custom branch filter is specified, so GitHub's default event behavior applies to branches covered by these events.

### 4.3 Execution environment

- Runner: `ubuntu-latest` (GitHub-hosted).
- Required permission: `contents: read`.

### 4.4 Jobs

#### Job 1: test
1. **Checkout repository**
   - Checks out the commit associated with the triggering event.
2. **Set up Go**
   - Installs Go `1.26`.
   - Enables the `actions/setup-go` cache.
3. **Verify Go version**
   - Runs `go version`.
4. **Run tests**
   - Executes `go test ./...`.
   - The workflow fails if the command exits with a non-zero status.

#### Job 2: lint
1. **Checkout repository**
   - Checks out the commit associated with the triggering event.
2. **Set up Go**
   - Installs Go `1.26`.
   - Enables the `actions/setup-go` cache.
3. **Run golangci-lint**
   - Executes `golangci/golangci-lint-action@v6` with `version: latest`.
   - Runs `golangci-lint run ./...` with default configuration.
   - The workflow fails if lint issues are found.

### 4.5 Acceptance criteria

- A push or pull request triggers the workflow.
- The workflow checks out the relevant source revision.
- Go is configured at version `1.26`.
- All Go package tests are executed using `go test ./...`.
- golangci-lint is executed against all Go packages.
- A test failure or lint issue marks the workflow run as failed.
- The workflow does not build or deploy a Docker image.
- The workflow does not require access to the self-hosted runner.

## 5. Workflow B: Build Webtester

### 5.1 Workflow file

`.github/workflows/build.yml`

### 5.2 Trigger conditions

- Trigger: `workflow_dispatch` only.
- The user manually starts the workflow from GitHub Actions.
- The user selects the branch using GitHub's native **Run workflow** branch selector.
- The user provides a required `version` string, for example `1.0.0`.

The workflow checks out `${{ github.ref_name }}`, which represents the selected ref name for the workflow run.

### 5.3 Execution environment

- Runner labels: `[self-hosted, Linux, X64]`.
- Intended host: `oong-serv`.
- Required GitHub Actions permission: `contents: read`.
- Podman must be available to the runner user.

### 5.4 Inputs

| Input     | Type   | Required | Description                                  |
| --------- | ------ | -------: | -------------------------------------------- |
| `version` | string |      Yes | Container image version/tag, such as `1.0.0`. |

The workflow validates the version against this pattern:

`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`

Invalid values cause the workflow to fail before the container image is built.

### 5.5 Steps

1. **Checkout selected branch**
   - Checks out the branch/ref selected in GitHub's workflow UI.
   - Uses `persist-credentials: false`.
2. **Validate image version**
   - Validates the supplied version string against pattern.
3. **Build container image**
   - Runs `podman build -t "webtester:${VERSION}" .`.
   - Uses the checked-out repository root as the build context.
   - Assumes the `Dockerfile` is in the repository root.
4. **Show build summary**
   - Records ref, commit SHA, and image tag in `$GITHUB_STEP_SUMMARY`.

### 5.6 Output

On successful completion, the container image is available in the local Podman engine used by the self-hosted runner, tagged as:

`webtester:<version>`

Example:

`webtester:1.0.0`

### 5.7 Acceptance criteria

- The workflow can be started manually from GitHub Actions.
- The user can select a branch/ref through GitHub's native branch selector.
- The user must provide a version.
- Invalid versions fail before the build step.
- The selected revision is checked out into the runner's workspace.
- The container image is built with the requested version tag.
- The workflow does not push the image to a registry.
- The workflow does not restart, replace, or deploy a running Webtester container.
- The workflow does not require SSH credentials to access `oong-serv`, because the job runs on that host's self-hosted runner.

## 6. Workflow C: Deploy Webtester

### 6.1 Workflow file

`.github/workflows/deploy.yml`

### 6.2 Trigger conditions

- Trigger: `workflow_dispatch` only.
- The user manually starts the workflow from GitHub Actions.
- The user selects the branch using GitHub's native **Run workflow** branch selector.
- The user provides a required `version` string matching a previously built image.

The workflow checks out `${{ github.ref_name }}`, which represents the selected ref name for the workflow run.

### 6.3 Execution environment

- Runner labels: `[self-hosted, Linux, X64]`.
- Intended host: `oong-serv`.
- Required GitHub Actions permission: `contents: read`.
- Podman must be available to the runner user.

### 6.4 Inputs

| Input     | Type   | Required | Description                                  |
| --------- | ------ | -------: | -------------------------------------------- |
| `version` | string |      Yes | Container image version to deploy (must exist from Build workflow). |

The workflow validates the version against this pattern:

`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`

### 6.5 Config Configuration via GitHub Actions

Configuration is managed completely via GitHub Repository Settings (**Settings → Secrets and variables → Actions**).

#### Variables (plain text):
- `HTTP_ADDR` (default: `:9090`)
- `DB_PATH` (default: `/data/webtester.db`)
- `ZAP_IMAGE` (default: `ghcr.io/zaproxy/zaproxy:stable`)
- `ZAP_TIMEOUT_SECONDS` (default: `1800`)
- `ZAP_RUNTIME` (default: `podman`)
- `REDIS_ADDR` (default: `redis:6379`)
- `REDIS_DB` (default: `0`)
- `AUTH_ACCESS_TTL` (default: `15m`)
- `AUTH_REFRESH_TTL` (default: `168h`)

#### Secrets (encrypted):
- `AUTH_ACCESS_SECRET`
- `AUTH_REFRESH_SECRET`
- `REDIS_PASSWORD`

### 6.6 Steps

1. **Checkout selected branch**
   - Checks out the branch/ref selected in GitHub's workflow UI.
   - Uses `persist-credentials: false`.
2. **Verify image exists**
   - Checks `podman image exists "webtester:<version>"`.
   - Fails if image not found with instruction to run Build workflow first.
3. **Generate `config.yaml`**
   - Generates production `config.yaml` dynamically from GitHub Variables & Secrets.
4. **Deploy with Podman Compose**
   - Runs `podman-compose up -d --remove-orphans` with `APP_VERSION=<version>`.
   - Starts or updates container using `webtester:<version>` image with mounted `config.yaml`.
5. **Verify container health**
   - Waits up to 60s for container to be running and `/healthz` endpoint to return 200.
6. **Show deployment summary**
   - Records version and timestamp in `$GITHUB_STEP_SUMMARY`.

### 6.7 Acceptance criteria

- The workflow can be started manually from GitHub Actions.
- The user can select a branch/ref through GitHub's native branch selector.
- The user must provide a version matching a previously built image.
- Invalid or missing images fail before deploy step with clear message.
- The container is deployed with the requested version tag.
- Container health is verified after deployment.
- The workflow does not push the image to a registry.
- The workflow does not require SSH credentials to access `oong-serv`.

## 6. Repository and Workspace Handling

The server does **not** need a manually maintained clone of the repository for these workflows.

`actions/checkout` checks out the repository into the GitHub Actions runner workspace before the steps that need the source code run. The exact workspace path depends on the runner configuration and should not be hardcoded.

The test and build jobs/workflows perform their own checkout. The test workflow uses GitHub-hosted infrastructure; the build workflow checks out the code on the self-hosted runner.

## 7. Workflow Independence

The test, build, and deploy workflows are independent.

- A successful test run does not automatically authorize or trigger a build.
- A failed test run does not, by itself, prevent a user from manually starting the build workflow.
- The build workflow does not download or reuse test artifacts; it checks out the selected source revision independently.
- The deploy workflow requires a previously built image but does not require the build workflow to have run in the same session. A user can build one day and deploy days later.
- The deploy workflow does not download or reuse build artifacts; it verifies image existence in local Podman storage.

If build eligibility must depend on successful tests for the exact same commit, an additional coordination or verification mechanism must be specified and implemented.

## 8. Security Requirements

- Use the minimum required GitHub Actions permission (`contents: read`).
- Do not place secrets or credentials directly in workflow files.
- Do not configure SSH credentials for the current build workflow; SSH is unnecessary when the runner is on the target host.
- Restrict administrative access to the self-hosted runner host and its container runtime (Podman/Docker). Container runtime access can grant substantial control over the host.
- Keep the self-hosted runner updated and limit which repositories and workflows are permitted to use it.
- Review third-party actions and pin action versions according to the repository's security policy.

## 9. Operational Requirements

Before using the build and deploy workflows, verify on `oong-serv` that:

- The self-hosted runner service is online.
- The runner has the expected labels.
- `podman ps` works under the same operating-system user that runs the runner.
- The Podman daemon is running.
- The repository's Dockerfile builds successfully with the provided build context.
- The Podman socket is available for container management (e.g. `systemctl --user enable --now podman.socket`).

The built image remains in the Podman engine used by the runner. If the runner is later moved to another machine, the image will be built on that machine instead. Previously built images are available for deployment without rebuild.

## 10. Non-Functional Requirements

- **Maintainability:** testing, image building, and deployment are defined in separate workflow files.
- **Traceability:** build summary records ref, commit SHA, and image tag; deploy summary records version and timestamp.
- **Repeatability:** a user can select a source ref and specify an explicit image version; deploy can target any previously built version.
- **Least privilege:** workflows request read-only repository contents permission.
- **Failure visibility:** failed tests, invalid version input, missing image, or build/deploy failures cause the corresponding workflow run to fail.
- **Rollback capability:** any previously built version can be deployed instantly without rebuild.

## 11. Known Limitations and Future Enhancements

The current design does not provide:

- Automatic deployment or container restart.
- Image registry publishing or remote image distribution.
- Automatic enforcement that tests pass before building.
- Protection against reusing an existing version tag in the local Docker engine.
- Automated application health checks or rollback after deployment.

Possible future enhancements include:

- Add a deployment workflow that runs only after explicit approval.
- Add a registry push step if images need to be distributed to other hosts.
- Add a check that verifies the exact commit passed tests before allowing a build.
- Add a deployment health check and rollback strategy.
- Establish a version-tagging convention and prevent accidental reuse of release versions.

## 12. Definition of Done

The implementation is considered complete when:

- [X] `.github/workflows/test.yml` runs Go tests on pushes and pull requests.
- [X] `.github/workflows/test.yml` runs golangci-lint on pushes and pull requests.
- [X] `.github/workflows/build.yml` can be manually triggered.
- [X] The build workflow supports GitHub's native branch selector.
- [X] The build workflow requires and validates an image version.
- [X] The build workflow creates `webtester:<version>` on the self-hosted runner.
- [X] `.github/workflows/deploy.yml` can be manually triggered.
- [X] The deploy workflow verifies image exists before deploying.
- [X] The deploy workflow generates `config.yaml` from GitHub Variables/Secrets.
- [X] The deploy workflow health-checks container after deployment.
- [X] Neither workflow performs an unrequested deployment or registry push.
- [X] All three workflows have been tested with a successful run and an expected failure scenario.
