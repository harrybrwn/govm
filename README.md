# govm

Go version manager.

## Install

```bash
go install github.com/harrybrwn/govm
```

## Usage

Download a version of go.
```bash
govm download 1.19.3
```

List all downloaded versions of go.
```bash
govm ls
```

Manually select a version.
```bash
gvm use 1.19.3
```

Select a version using a config file.
```bash
echo '1.18.5' > .govm
govm use
```

