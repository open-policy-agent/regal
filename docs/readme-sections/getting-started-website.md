<!-- markdownlint-disable MD041 MD033 -->

## Getting Started

### Installing Regal

This section explains how you can install and run Regal on your own machine.

<Tabs queryString="current-os">
<TabItem value="macos" label="macOS" default>

<Tabs>
  <TabItem value="brew" label="Homebrew" default>
    Regal binaries can be installed on macOS using Homebrew. The formula can be
    reviewed on [brew.sh](https://formulae.brew.sh/formula/regal). This method
    supports both ARM64 and AMD64 architectures.
    ```shell
    brew install regal
    ```
  </TabItem>
  <TabItem value="mise" label="mise">
    If you are using [mise](https://mise.jdx.dev), the polyglot tool version manager, you can install Regal using:
    ```shell
    mise use -g regal@latest
    ```
  </TabItem>
</Tabs>

**Manual Download**

It's also possible to download the Regal binary directly:

<Tabs>
  <TabItem value="arm64" label="arm64 (Apple Silicon)" default>
    ```shell
    curl -L -o regal https://github.com/open-policy-agent/regal/releases/latest/download/regal_Darwin_arm64
    ```
  </TabItem>
  <TabItem value="amd64" label="amd64 (Older Intel Macs)">
    ```shell
    curl -L -o regal https://github.com/open-policy-agent/regal/releases/latest/download/regal_Darwin_x86_64
    ```
  </TabItem>
</Tabs>

After downloading the Regal binary, you must ensure it's executable:

```shell
chmod 755 ./regal
```

It's also recommended to move the Regal binary into a directory in your
`PATH` so you can run Regal commands in different directories.

You can verify the installation by running:

```shell
regal version
```

</TabItem>

<TabItem value="linux" label="Linux/Unix">
In order to manually install the Regal binary from the GitHub release assets,
please run the following:

<Tabs>
  <TabItem value="linux_arm64" label="arm64" default>
    ```shell
    curl -L -o regal https://github.com/open-policy-agent/regal/releases/latest/download/regal_Linux_arm64
    ```
  </TabItem>
  <TabItem value="linux_amd64" label="amd64">
    ```shell
    curl -L -o regal https://github.com/open-policy-agent/regal/releases/latest/download/regal_Linux_x86_64
    ```
  </TabItem>
</Tabs>
After downloading the Regal binary, you must ensure it's executable:
```shell
chmod 755 ./regal
```
It's also recommended to move the Regal binary into a directory in your
`PATH` so you can run Regal commands in any directory.

You can verify the installation by running:

```shell
regal version
```

:::info Community Package Repositories
There are a number of community-maintained package repositories that provide Regal binaries for Linux/Unix.

See the [Packaging section](https://www.openpolicyagent.org/projects/regal/adopters#packaging)
of the Adopters page for a complete list of available package managers.

These packages are maintained by their respective communities and may not always have the latest Regal version available.
:::

</TabItem>

<TabItem value="windows" label="Windows">
Download the Windows binary using PowerShell:

```powershell
Invoke-WebRequest -Uri "https://github.com/open-policy-agent/regal/releases/latest/download/regal_Windows_x86_64.exe" -OutFile "regal.exe"
```

Or using curl (if available):

```cmd
curl -L -o regal.exe https://github.com/open-policy-agent/regal/releases/latest/download/regal_Windows_x86_64.exe
```

Add the Regal binary to your PATH by creating a Tools directory for it:

```cmd
mkdir C:\Tools\Regal
move regal.exe C:\Tools\Regal\
```

Now we can add this to our `PATH`:

Control Panel → System → Advanced system settings → Environment Variables

Edit the Path variable → Add: `C:\Tools\Regal`

Alternatively, run:

```powershell
[Environment]::SetEnvironmentVariable("Path", "$env:Path;C:\Tools\Regal", "User")
```

You can verify the installation by running:

```cmd
regal version
```

</TabItem>
<TabItem value="docker" label="Docker">
You can also download and run Regal via Docker. The latest stable image tag is
`ghcr.io/open-policy-agent/regal:latest`.

You can verify the installation by running:

```shell
docker run --rm ghcr.io/open-policy-agent/regal:latest version
```

</TabItem>
</Tabs>

See all available binaries on the
[GitHub releases](https://github.com/open-policy-agent/regal/releases) page.
Checksums for all binaries are available in the download path by appending
`.sha256` to the binary filename.

For example, verify the macOS arm64 binary checksum:

```shell
BINARY_NAME=regal_Darwin_arm64
curl -L -O https://github.com/open-policy-agent/regal/releases/latest/download/$BINARY_NAME
curl -L -O https://github.com/open-policy-agent/regal/releases/latest/download/$BINARY_NAME.sha256
shasum -c $BINARY_NAME.sha256
```
