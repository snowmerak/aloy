# aloy

C++ Package Manager & Meta Build System

**aloy** is a Go-based tool that lets you manage dependencies and build C++ projects with a single YAML configuration, without writing CMakeLists.txt by hand.

## Core Philosophy

- **CMake Abstraction** — Just maintain `project.yaml` and Modern CMake (target-based) is generated automatically
- **Local Isolation** — Dependencies are stored in `.my_modules/`, preventing global environment pollution and ABI conflicts
- **Version Conflict Detection** — Raises an error when the same package is required with different major versions, recommending `alias` as a workaround
- **Legacy Compatible** — Existing open-source repositories with their own CMakeLists.txt work out of the box

## Installation

```bash
go install github.com/snowmerak/aloy@latest
```

### Prerequisites

The following tools must be available on your system PATH:

- `git`
- `cmake` (>= 3.15)
- A C++ compiler (GCC, Clang, MSVC, etc.)
- `meson` and `ninja` (required if utilizing Meson dependencies)

## Quick Start

```bash
# Initialize an executable project
mkdir my_project && cd my_project
aloy init

# Initialize a library project
aloy init --type library

# Add dependencies
aloy add https://github.com/gabime/spdlog.git --cmake-option SPDLOG_BUILD_TESTS=OFF
aloy add https://github.com/nlohmann/json.git --cmake-target nlohmann_json
aloy add OpenSSL --system

# Sync (resolve dependencies + generate CMake + cmake configure)
aloy sync

# Build
aloy build

# Build and run the executable
aloy run

# Build and run tests via CTest
aloy test
```

## Project Structure

```
my_project/
├── project.yaml             # Project definition and dependency spec
├── aloy.lock                # Pinned dependency versions and commit hashes
├── src/                     # Source code
├── tests/                   # [Auto] Test code
├── include/                 # Public headers
├── .my_modules/             # [Auto] Downloaded package sources
├── build/                   # [Auto] CMake build output
└── CMakeLists.txt           # [Auto] Generated master CMake script
```

## project.yaml

```yaml
project:
  name: MyStreamingServer
  version: 1.0.0
  cxx_standard: 17

targets:
  mediaserver:
    type: executable           # executable, library, shared_library, header_only, test
    pch: "include/pch.h"       # [Optional] Precompiled header
    sources:
      - "src/main.cpp"
      - "src/core/**/*.cpp"    # glob patterns supported
    includes:
      public: ["include/"]
      private: ["src/core/private"]

    platforms:
      linux:
        sources: ["src/platform/linux/**/*.cpp"]
        compiler_flags: ["-O2", "-g"]
      windows:
        sources: ["src/platform/win/**/*.cpp"]
        compiler_flags: ["/wd4819"]

    dependencies:
      - name: logger
        git: "git@github.com:company/logger.git"
        version: "^1.2.0"
        alias: alog

      - name: OpenSSL
        type: system

      - name: spdlog
        git: "https://github.com/gabime/spdlog.git"
        cmake_options:
          SPDLOG_BUILD_TESTS: "OFF"

      - name: json
        git: "https://github.com/nlohmann/json.git"
        cmake_target: nlohmann_json    # When CMake target name differs from package name

      # Monorepo support via subdir
      - name: core_utils
        git: "https://github.com/company/monorepo.git"
        subdir: libs/core_utils
        version: "^1.0.0"

inject_cmake: "cmake/extra_logic.cmake"
```

### Dependency Fields

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Package name (auto-inferred from Git URL) |
| `git` | Cond. | Git repository URL (not needed for system type) |
| `version` | No | SemVer constraint (`^1.2.0`, `~1.0`, `>=1.0.0 <2.0.0`) |
| `type` | No | Dependency type (`"system"`, `"cmake"`, `"meson"`, `"aloy"`, `"header_only"`). Omitted defaults to auto-detecting based on the files in repository. |
| `alias` | No | Alias — used as the directory name and reference name |
| `subdir` | No | Subdirectory path for monorepo packages (e.g. `libs/foo`) |
| `cmake_target` | No | CMake target name for linking (when it differs from the package name) |
| `cmake_options` | No | `key: value` map — injected as `set(KEY VAL CACHE ... FORCE)` |

> **When is `cmake_target` needed?**
> Some libraries have CMake target names that differ from their repository name.
> For example, the `nlohmann/json` repository exposes the CMake target `nlohmann_json`, not `json`.
> In such cases, specify `cmake_target: nlohmann_json` so that `target_link_libraries` links correctly.

## CLI Commands

| Command | Description |
|---|---|
| `aloy init` | Initialize project (`project.yaml`, `src/`, `tests/`, `include/`) |
| `aloy add <url\|name>` | Add a dependency |
| `aloy remove <name>` | Remove a dependency (searches by name or alias) |
| `aloy sync` | Resolve dependencies → generate CMake → run `cmake -B build` |
| `aloy build` | Build the project (auto-runs sync if needed) |
| `aloy run` | Build and run an executable target |
| `aloy test` | Build and run testing targets (`type: test`) via CTest |
| `aloy tree` | Show the resolved dependency hierarchy tree |
| `aloy clean` | Remove `build/` (`--all` removes `.my_modules/`, `--cache` clears `~/.aloy/cache`) |
| `aloy download` | Clone sources from `aloy.lock` only (for CI) |
| `aloy update` | Update to latest versions within SemVer range (`--dry-run` supported) |

### init Options

```bash
aloy init                    # Executable project (default)
aloy init --type library     # Static library project
aloy init -t shared_library  # Shared library project
aloy init -t header_only     # Header-only library project
```

### add Options

```bash
aloy add <git_url>                              # Basic add
aloy add <git_url> -v "^1.2.0"                 # Version constraint
aloy add <git_url> -a myalias                  # Set alias
aloy add <name> --system                        # System package (find_package)
aloy add <git_url> --cmake-option KEY=VAL       # CMake option (repeatable)
aloy add <git_url> --cmake-target target_name   # CMake target name override
aloy add <git_url> --subdir path/to/pkg         # Monorepo subdirectory
aloy add <git_url> --type <type>                # Explicitly set type (system, cmake, meson, aloy)
aloy add <git_url> -t mytarget                  # Add to a specific target
```

### build Options

```bash
aloy build                    # Release build (default)
aloy build --config Debug     # Debug build
aloy build -j 8               # Parallel build
```

### clean Options

```bash
aloy clean                    # Remove build/ only
aloy clean --all              # Remove build/ + .my_modules/ + CMakeLists.txt
aloy clean --cache            # Clear global git cache in ~/.aloy/cache
```

### update Options

```bash
aloy update                   # Update all deps to latest within range
aloy update --dry-run         # Show what would change without applying
```

## Dependency Resolution

aloy resolves dependencies using **BFS (breadth-first search)**:

1. All dependencies from every target are queued and processed in order
2. Dependencies are cloned centrally into a **Global Git Cache** (`~/.aloy/cache`) via bare clone to eliminate duplicate network IO
3. For each dependency, SemVer-compatible versions are discovered from Git tags (`v1.2.0` → `1.2.0`)
4. SemVer ranges are supported: `^1.2.0`, `~1.2.0`, `>=1.0.0 <2.0.0`, etc.
5. `aloy` then rapidly creates a hardlinked copy (`--reference`) in `.my_modules/<repo-hash-version>/`
6. When the same package is required multiple times, the **first resolved version is kept** (first-come-first-served)
7. If **major versions conflict**, an error is raised with a hint to use `alias`
8. Repositories without matching tags fall back to the default branch (`main` → `master`)
9. If a dependency is an aloy package (has `project.yaml`), its sub-dependencies are resolved recursively

### Meson Integration

When `aloy` detects a dependency of type `"meson"` (either explicitly specified via `type: meson` or auto-detected by the presence of `meson.build` in the repository's root/subdirectory):

1. During `aloy sync`, it executes `meson setup build --prefix=<install_path>`, `meson compile`, and `meson install` to build and stage the package locally.
2. The compiled outputs (static/shared libraries and headers) are gathered inside `.my_modules/<logical_name>_install/`.
3. In the root `CMakeLists.txt`, an `INTERFACE IMPORTED` target is generated. All compiled libraries found in the staging area are linked via `INTERFACE_LINK_LIBRARIES` and headers are linked via `INTERFACE_INCLUDE_DIRECTORIES`.

### Header-only Integration

When `aloy` detects a dependency of type `"header_only"` (either explicitly specified via `type: header_only` or automatically detected when no `project.yaml`, `meson.build`, or `CMakeLists.txt` exist in the repository):

1. During `aloy sync`, it skips all compilation/build stages entirely.
2. It automatically scans the repository for common header directories (specifically searching for `single_include`, `include`, `includes`, `src`, or falling back to the root directory).
3. In the root `CMakeLists.txt`, `aloy` generates an `INTERFACE IMPORTED` target and maps the discovered header folder path under `INTERFACE_INCLUDE_DIRECTORIES`.

### Transitive Dependencies and `cmake_target`

aloy sub-packages (packages in `.my_modules/` that have their own `project.yaml`) are resolved recursively. However, **a sub-package's `cmake_target` setting is not propagated to the parent project.**

For example:
- Package A depends on `nlohmann/json` and sets `cmake_target: nlohmann_json`
- If the parent project also directly uses `nlohmann/json`, it must specify `cmake_target: nlohmann_json` separately

This is because each project's `project.yaml` is managed independently. Transitive dependencies' CMake target names are resolved by the original CMakeLists.txt defined in that package.

## CMake Generation

- Source files are explicitly listed instead of using `file(GLOB)`
- Uses target-based commands: `target_include_directories`, `target_link_libraries`, etc.
- Generates platform-specific conditional blocks: `if(WIN32)` / `if(UNIX AND NOT APPLE)` / `if(APPLE)` / `if(UNIX)`
- `type: system` packages are handled via `find_package()`
- `cmake_options` are injected as `set(VAR VAL CACHE STRING "" FORCE)`
- Dependencies with `cmake_target` set are linked using that name in `target_link_libraries`
- Custom CMake logic can be included via `inject_cmake`
- Separate CMakeLists.txt files are generated for aloy sub-packages inside `.my_modules/`

### Link Name Priority

The name used in `target_link_libraries` is determined by this priority:

1. `type: system` → the package's `name`
2. `cmake_target` is set → the `cmake_target` value
3. Otherwise → `alias` if set, else `name`

## Target Types

| Type | CMake Mapping | Description |
|---|---|---|
| `executable` | `add_executable()` | Executable binary |
| `library` | `add_library(STATIC)` | Static library |
| `shared_library` | `add_library(SHARED)` | Shared / dynamic library |
| `header_only` | `add_library(INTERFACE)` | Header-only library (no sources required) |
| `test` | `add_test() / add_executable()` | Test binary to be run via CTest |

## Known Limitations

- **No explicit cycle detection** — Circular dependencies do not cause infinite loops (the `resolved` map prevents re-processing), but they are silently accepted without warning
- **No SemVer intersection** — When the same package is required with different version ranges, the first resolved version is used (no intersection computation)
- **Transitive `cmake_target` not propagated** — See the "Transitive Dependencies and `cmake_target`" section above

## License

MIT
