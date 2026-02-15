# Test Project for GolangZakhireh

This directory contains a simple Go project used to verify the functionality of the GolangZakhireh proxy.

## Purpose

The `test-project` imports several external dependencies (e.g., `google/uuid`, `sirupsen/logrus`, `pkg/errors`) to trigger module downloads. When you build or run this project while configured to use GolangZakhireh, it validates that:

1.  GolangZakhireh can successfully fetching modules from the upstream proxy.
2.  GolangZakhireh caches these modules locally.
3.  GolangZakhireh can serve these modules from the cache in subsequent runs or offline mode.

## Usage

### 1. Start GolangZakhireh

Make sure the GolangZakhireh server is running on port 8811.

```bash
go run ./cmd/server
```

### 2. Configure Environment

Set `GOPROXY` to point to your local GolangZakhireh instance.

```bash
export GOPROXY=http://localhost:8811/proxy,direct
export GOSUMDB=off # Optional, if you encounter checksum issues with private modules or simple testing
```

### 3. Run the Test Project

Navigate to this directory and run the project.

```bash
cd test-project
go run main.go
```

### 4. Verify

-   Check the output of `go run`. It should print "GolangZakhireh test project starting..." and other logs.
-   Check the GolangZakhireh dashboard at `http://localhost:8811`. You should see the modules (`github.com/google/uuid`, etc.) listed in the cache.
