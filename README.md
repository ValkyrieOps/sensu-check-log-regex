[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/ValkyrieOps/sensu-check-log-regex)
![Go Test](https://github.com/ValkyrieOps/sensu-check-log-regex/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/ValkyrieOps/sensu-check-log-regex/workflows/goreleaser/badge.svg)

# sensu-check-log-regex

## Table of Contents
- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

The sensu-check-log-regex is a [Sensu Check][6] that uses regex to look at nested directories and return all log files and their respective matches.  It is based on the original [sensu-check-log](https://github.com/sensu/sensu-check-log) check, but is not at feature parity, does not use the event API, and uses the Sensu Go Asset plugin format for arguments.  In addition this check scrubs ':' from paths to prevent issues with Windows paths. 

## Files
- bin/sensu-check-log-regex

## Usage examples
```
Usage:
  sensu-check-log-regex [flags]
  sensu-check-log-regex [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -h, --help              help for sensu-check-log-regex
  -l, --logpath string    Path of logs to examine
  -r, --logregex string   Regex of log names to examine
  -m, --match string      Keyword to match in logs
  -n, --numprocs int      Number of processors to use (defaults to runtime.NumCPU())
  -s, --state string      Path to root state directory

Use "sensu-check-log-regex [command] --help" for more information about a command.
```

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add ValkyrieOps/sensu-check-log-regex
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/ValkyrieOps/sensu-check-log-regex].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-check-log-regex
  namespace: default
spec:
  command: sensu-check-log-regex -l "/tmp/test/logs" -r "*.txt" -m "ERROR" -s "/tmp/test/state"
  subscriptions:
  - system
  runtime_assets:
  - ValkyrieOps/sensu-check-log-regex
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-check-log-regex repository:

```
go build
```

## Additional notes
The default for -numprocs is determined by [runtime.NumCPU()](https://golang.org/pkg/runtime/#NumCPU).
>NumCPU returns the number of logical CPUs usable by the current process. The set of available CPUs is checked by querying the operating system at process startup. Changes to operating system CPU allocation after process startup are not reflected.

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://github.com/sensu-community/sensu-plugin-sdk
[3]: https://github.com/sensu-plugins/community/blob/master/PLUGIN_STYLEGUIDE.md
[4]: https://github.com/sensu-community/check-plugin-template/blob/master/.github/workflows/release.yml
[5]: https://github.com/sensu-community/check-plugin-template/actions
[6]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[7]: https://github.com/sensu-community/check-plugin-template/blob/master/main.go
[8]: https://bonsai.sensu.io/
[9]: https://github.com/sensu-community/sensu-plugin-tool
[10]: https://docs.sensu.io/sensu-go/latest/reference/assets/
