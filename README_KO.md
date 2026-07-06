# aloy

C++ 패키지 매니저 & 메타 빌드 시스템

**aloy**는 복잡한 CMakeLists.txt를 직접 작성하지 않고, YAML 설정 하나로 의존성 관리부터 빌드까지 한 번에 수행할 수 있게 돕는 Go 기반 도구입니다.

## 핵심 철학

- **CMake 추상화** — `project.yaml`만 관리하면 Modern CMake (Target-based)가 자동 생성됩니다
- **로컬 격리** — 의존성은 `.my_modules/`에 격리 저장하여 전역 환경 오염과 ABI 충돌을 방지합니다
- **버전 충돌 감지** — 동일 패키지에 대해 메이저 버전이 다르면 에러를 발생시키고 `alias` 사용을 권고합니다
- **레거시 호환** — 기존 CMakeLists.txt가 있는 일반 오픈소스 저장소도 그대로 사용할 수 있습니다

## 설치

```bash
go install github.com/snowmerak/aloy@latest
```

### 전제 조건

시스템 PATH에 다음 도구가 필요합니다:

- `git`
- `cmake` (>= 3.15)
- C++ 컴파일러 (GCC, Clang, MSVC 등)
- `meson` 및 `ninja` (Meson 의존성을 사용하는 경우 필요)

## 빠른 시작

```bash
# 실행 파일 프로젝트 초기화
mkdir my_project && cd my_project
aloy init

# 라이브러리 프로젝트 초기화
aloy init --type library

# 의존성 추가
aloy add https://github.com/gabime/spdlog.git --cmake-option SPDLOG_BUILD_TESTS=OFF
aloy add https://github.com/nlohmann/json.git --cmake-target nlohmann_json
aloy add OpenSSL --system

# 동기화 (의존성 해석 + CMake 생성 + cmake configure)
aloy sync

# 빌드
aloy build

# 실행 파일 빌드 및 즉시 실행
aloy run

# CTest를 통해 통합 테스트 실행
aloy test
```

## 프로젝트 구조

```
my_project/
├── project.yaml             # 프로젝트 정의 및 의존성 명세
├── aloy.lock                # 고정된 의존성 버전 및 커밋 해시
├── src/                     # 소스 코드
├── tests/                   # [자동] 테스트 코드
├── include/                 # 공용 헤더
├── .my_modules/             # [자동] 다운로드된 패키지 소스
├── build/                   # [자동] CMake 빌드 결과물
└── CMakeLists.txt           # [자동] aloy가 생성한 마스터 스크립트
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
    pch: "include/pch.h"       # [선택사항] 사전 컴파일된 헤더
    sources:
      - "src/main.cpp"
      - "src/core/**/*.cpp"    # glob 패턴 지원
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
        cmake_target: nlohmann_json    # CMake 타겟명과 패키지명이 다를 경우

      # 모노레포 지원 (subdir 옵션)
      - name: core_utils
        git: "https://github.com/company/monorepo.git"
        subdir: libs/core_utils
        version: "^1.0.0"

inject_cmake: "cmake/extra_logic.cmake"
```

### 의존성 필드

| 필드 | 필수 | 설명 |
|---|---|---|
| `name` | O | 패키지 이름 (Git URL에서 자동 추론 가능) |
| `git` | △ | Git 저장소 URL (system 타입이면 불필요) |
| `version` | No | SemVer 제약 조건 (`^1.2.0`, `~1.0`, `>=1.0.0 <2.0.0`) |
| `type` | No | 의존성 빌드 타입 (`"system"`, `"cmake"`, `"meson"`, `"aloy"`). 생략 시 파일 구성을 기반으로 자동 감지합니다. |
| `alias` | No | 모듈로 저장되는 별칭 (이름 중복 방지) |
| `subdir` | No | 모노레포 패키지를 위한 하위 디렉토리 명시 (예: `libs/foo`) |
| `cmake_target` | No | `target_link_libraries`에 사용될 타겟명 오버라이드 |
| `cmake_options` | No | `키: 값` 쌍 — CMake CACHE 변수로 주입 |`set(KEY VAL CACHE ... FORCE)` 로 주입 |

> **`cmake_target`은 언제 필요한가?**
> 일부 라이브러리는 CMake 타겟 이름이 저장소 이름과 다릅니다.
> 예: `nlohmann/json` 저장소의 CMake 타겟은 `nlohmann_json::nlohmann_json`이 아닌 `nlohmann_json`입니다.
> 이런 경우 `cmake_target: nlohmann_json`을 지정해야 `target_link_libraries`에서 올바르게 링킹됩니다.

## CLI 명령어

| 명령어 | 설명 |
|---|---|
| `aloy init` | 프로젝트 초기화 (`project.yaml`, `src/`, `tests/`, `include/` 생성) |
| `aloy add <url\|name>` | 의존성 추가 |
| `aloy remove <name>` | 의존성 제거 (이름 또는 별칭으로 검색) |
| `aloy sync` | 의존성 해석 → CMake 생성 → `cmake -B build` 실행 |
| `aloy build` | 빌드 수행 (필요 시 sync 자동 호출) |
| `aloy run` | 실행 파일 타겟을 빌드하고 즉시 실행 |
| `aloy test` | (`type: test`) 타겟을 빌드하고 `CTest`로 테스트 진행 |
| `aloy tree` | 해결된 의존성 계층 구조 확인 |
| `aloy clean` | `build/` 폴더 제거 (`--all` 옵션 시 `.my_modules/` 제거, `--cache` 옵션 시 `~/.aloy/cache` 전역 캐시 비우기) |
| `aloy download` | 평가 과정 없이 `aloy.lock`에 명시된 소스만 다운로드 (CI용) |
| `aloy update` | SemVer 제약 조건 내에서 가능한 최신 버전으로 업데이트 (`--dry-run` 지원) |

### init 옵션

```bash
aloy init                    # executable 프로젝트 (기본)
aloy init --type library     # 정적 라이브러리 프로젝트
aloy init -t shared_library  # 공유 라이브러리 프로젝트
aloy init -t header_only     # 헤더 온리 라이브러리 프로젝트
```

### add 옵션

```bash
aloy add <git_url>                              # 기본 추가
aloy add <git_url> -v "^1.2.0"                 # 버전 제약
aloy add <git_url> -a myalias                  # 별칭 지정
aloy add <name> --system                        # 시스템 패키지 추가
aloy add <git_url> --cmake-option KEY=VAL       # CMake 옵션 추가 (반복 가능)
aloy add <git_url> --cmake-target target_name   # CMake 타겟 이름 지정
aloy add <git_url> --subdir path/to/pkg         # 모노레포 하위 폴더 지정
aloy add <git_url> --type <type>                # 명시적 타입 지정 (system, cmake, meson, aloy)
aloy add <git_url> -t mytarget                  # 특정 타겟에만 추가
```

### build 옵션

```bash
aloy build                    # Release 빌드 (기본)
aloy build --config Debug     # Debug 빌드
aloy build -j 8               # 병렬 빌드
```

### clean 옵션

```bash
aloy clean                    # build/ 삭제
aloy clean --all              # build/, .my_modules/, 생성된 CMakeLists.txt 모두 삭제
aloy clean --cache            # ~/.aloy/cache 전역 캐시 삭제
```

### update 옵션

```bash
aloy update                   # 모든 의존성을 범위 내 최신으로 갱신
aloy update --dry-run         # 변경 사항만 출력하고 실제 갱신하지 않음
```

## 의존성 해석

aloy의 의존성 해석은 **BFS (너비 우선 탐색)** 기반으로 동작합니다:

1. 모든 타겟의 의존성이 단일 큐에 배치되어 순차적으로 처리됩니다.
2. 중복되는 네트워크 IO를 제거하기 위해 모든 저장소는 **전역 Git 캐시 (`~/.aloy/cache`)** 에 백업(bare clone)됩니다.
3. Git 태그를 가져와 SemVer 조건과 가장 잘 맞는 버전을 도출합니다. (예: `v1.2.0` → `1.2.0`)
4. SemVer 범위 제약을 지원합니다: `^1.2.0`, `~1.2.0`, `>=1.0.0 <2.0.0` 등.
5. 이후 `aloy`는 전역 캐시를 참조(`--reference`)하여 `.my_modules/<repo-hash-version>/` 위치에 0초 만에 코드를 전개합니다.
6. 복수의 동일 패키지가 요구된 경우 **가장 먼저 확인된 버전을 채택**합니다 (선착순 우선권).
7. 단, **메이저 버전 충돌**이 발생할 경우 `alias` 옵션을 사용하라는 힌트와 함께 오류를 중단합니다.
8. 조건과 일치하는 태그가 없으면 기본 브랜치(`master` 또는 `main`)로 fallback됩니다.
9. 동기화된 타겟이 aloy 패키지인 경우(`project.yaml` 존재) 서브 의존성을 재귀적으로 확보합니다.

### Meson 연동

aloy가 의존성 타입을 `"meson"`으로 감지할 경우(명시적으로 `type: meson`을 지정했거나, 저장소 루트/하위 폴더에 `meson.build` 파일이 존재할 경우):

1. `aloy sync` 실행 시 백그라운드에서 `meson setup build --prefix=<install_path>`, `meson compile`, `meson install`을 순차적으로 호출하여 의존성을 자동 빌드합니다.
2. 컴파일된 결과물(정적/동적 라이브러리 및 공용 헤더)은 로컬 스테이징 폴더 `.my_modules/<logical_name>_install/`에 수집됩니다.
3. 생성되는 마스터 `CMakeLists.txt` 내에 `add_library(<name> INTERFACE IMPORTED)` 타겟이 추가되며, 스테이징 영역에서 발견된 모든 라이브러리 파일이 `INTERFACE_LINK_LIBRARIES`를 통해 링크되고 헤더 파일은 `INTERFACE_INCLUDE_DIRECTORIES`로 매핑됩니다.

### 전이 의존성과 `cmake_target`

aloy 서브패키지(`.my_modules/` 내에서 `project.yaml`을 가진 패키지)는 재귀적으로 의존성이 해석됩니다. 하지만 **서브패키지의 `cmake_target` 설정은 상위 프로젝트로 전파되지 않습니다.**

예를 들어:
- 패키지 A가 `nlohmann/json`에 의존하고 `cmake_target: nlohmann_json`을 설정했더라도
- A를 사용하는 상위 프로젝트에서 직접 `nlohmann/json`도 사용한다면, 상위 프로젝트에서도 `cmake_target: nlohmann_json`을 별도로 지정해야 합니다

이는 각 프로젝트의 `project.yaml`이 독립적으로 관리되기 때문입니다. 전이 의존성의 CMake 타겟 이름은 해당 패키지의 원래 CMakeLists.txt가 정의하는 타겟으로 해석됩니다.

## CMake 생성

- `file(GLOB)` 대신 소스 파일 목록을 명시적으로 하드코딩합니다
- `target_include_directories`, `target_link_libraries` 등 타겟 기반 명령어를 사용합니다
- 플랫폼별 `if(WIN32)` / `if(UNIX AND NOT APPLE)` / `if(APPLE)` / `if(UNIX)` 조건 블록을 생성합니다
- `type: system` 패키지는 `find_package()` 로 처리합니다
- `cmake_options`는 `set(VAR VAL CACHE STRING "" FORCE)` 로 주입합니다
- `cmake_target`이 지정된 의존성은 해당 이름으로 `target_link_libraries`에 링킹됩니다
- `inject_cmake` 로 커스텀 CMake 로직을 삽입할 수 있습니다
- aloy 서브패키지에 대해서는 `.my_modules/` 내에 별도의 CMakeLists.txt를 생성합니다

### 링킹 우선순위

`target_link_libraries`에서 사용되는 이름의 결정 우선순위:

1. `type: system` → 패키지의 `name`
2. `cmake_target`이 지정됨 → `cmake_target` 값
3. 그 외 → `alias`가 있으면 `alias`, 없으면 `name`

## 타겟 타입

| 타입 | CMake 매핑 | 설명 |
|---|---|---|
| `executable` | `add_executable()` | 실행 파일 |
| `library` | `add_library(STATIC)` | 정적 라이브러리 |
| `shared_library` | `add_library(SHARED)` | 공유 라이브러리 |
| `header_only` | `add_library(INTERFACE)` | 헤더 온리 라이브러리 (소스 불필요) |
| `test` | `add_test() / add_executable()` | CTest로 구동되는 테스트용 바이너리 |

## 알려진 제한 사항

- **순환 의존성 미경고** — 순환 의존성이 있어도 `resolved` 맵으로 중복 처리를 방지하여 무한 루프는 발생하지 않지만, 순환이 감지되어도 경고 없이 무시됩니다
- **SemVer 교집합 미지원** — 동일 패키지에 대해 서로 다른 버전 범위가 요구되면 먼저 해석된 버전이 사용됩니다 (교집합 계산 없음)
- **전이 의존성 cmake_target 미전파** — 위 "전이 의존성과 cmake_target" 섹션 참조

## 라이선스

MIT
