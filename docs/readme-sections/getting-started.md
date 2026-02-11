<!-- markdownlint-disable MD041 -->

## Getting Started

### Download Regal

```shell
# macOS
brew install regal

# linux x86
curl -L -o regal https://github.com/open-policy-agent/regal/releases/latest/download/regal_Linux_x86_64
chmod 755 ./regal

# windows
Invoke-WebRequest -Uri "https://github.com/open-policy-agent/regal/releases/latest/download/regal_Windows_x86_64.exe" -OutFile "regal.exe"
```

<details>
  <summary><strong>Other Installation Options & Packages</strong></summary>

Manual installation commands:

**MacOS (Apple Silicon)**

```shell
curl -L -o regal "https://github.com/open-policy-agent/regal/releases/latest/download/regal_Darwin_arm64"
```

**MacOS (x86_64)**

```shell
curl -L -o regal "https://github.com/open-policy-agent/regal/releases/latest/download/regal_Darwin_x86_64"
```

**Linux (arm64)**

```shell
curl -L -o regal "https://github.com/open-policy-agent/regal/releases/latest/download/regal_Linux_arm64"
chmod 755 ./regal
```

**Docker**

```shell
docker pull ghcr.io/open-policy-agent/regal:latest
```

Please see [Packages](https://www.openpolicyagent.org/projects/regal/adopters#packaging)
for a list of package repositories which distribute Regal.

See all versions, and checksum files, at the Regal [releases](https://github.com/open-policy-agent/regal/releases/)
page, and published Docker images at the [packages](https://github.com/open-policy-agent/regal/pkgs/container/regal)
page.

</details>
