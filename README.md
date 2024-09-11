# spycat
An eBPF observable agent to solve os performance issue in kernel or app like bcc tool.
These tools developed refer to bcc which are really useful for solving os issue and some
new feature added.
* support more kernel version
* support data store with third-party database
* ...

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/chentao-kernel/spycat)](https://github.com/chentao-kernel/spycat/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/chentao-kernel/spycat)](https://goreportcard.com/report/github.com/chentao-kernel/spycat)
[![License](https://img.shields.io/github/license/chentao-kernel/spycat)](https://github.com/chentao-kernel/spycat/blob/main/LICENSE)

----

* [Compoents](#Components)
* [Feature](#Feature)
* [Quick Start](#Quick-Start)
* [License](#License)
* [Thanks](#Thanks)

## Components
<div align=center> <img src="doc/spycat.png" width = "60%" height="60%" /> </div>

## Feature List
### Tool List
#### cpu subsystem
- [X] `oncpu`
continus profiling to detect cpu burst issue, data can be stored in pyroscope
- [X] `offcpu`
detect task scheduling not timely issue, like web app timeout etc.
- [X] `futexsnoop`
detect multitasking lock contention issue
#### mem subsystem
#### io subsystem
#### net subsystem

### Exporter List
- [X] tmerminal
- [X] local storage
- [X] loki
- [X] pyroscope
- [ ] influxdb
- [ ] prometheus

## Quick Start
### How to build
You can use docker/build.sh to build in a container.
```
usage: ./args.sh [-h|--help -b|--build -c|--compile -t|--tar -V|--bin_ver]
        -b              build image
        -c              compile
        -t              tar binary
        -V 0.1.2 -t     tar binary with 0.1.2 version
```
### How to run
##### tool list help
```
SUBCOMMANDS
  completion  generate the autocompletion script for the specified shell
  futexsnoop  eBPF snoop user futex
  help        Help about any command
  offcpu      eBPF offcpu profiler
  oncpu       eBPF oncpu sampling profiler
  version     Print spycat version details
FLAGS         DEFAULT VALUES
  --help     false
    help for spycat
  --version  false
Run 'Spycat SUBCOMMAND --help' for more information on a subcommand.
```
##### futexsnoop for example
```
sudo ./spycat futexsnoop -h
FLAGS                     DEFAULT VALUES
  --app-name             
    application name used when uploading profiling data
  --help                 false
    help for futexsnoop
  --log-level            info
    log level: debug|info|warn|error
  --max-dur-ms           1000000
    max time(ms) wait unlock
  --max-lock-hold-users  100
    max users hold the same lock
  --min-dur-ms           1000
    min time(ms) wait unlock
  --pid                  0
    pid to trace, -1 to trace all pids
  --stack                false
    get stack info or not
  --symbol-cache-size    256
    max size of symbols cache
  --target-lock          0
    target lock addr
  --tid                  0
    tid to trace, -1 to trace all tids
```
### How to develop
## License
spycat is distributed under [Apache License, Version2.0]
## Acknowledgements
This project makes use of the following open source projrces:
- [kindling](https://github.com/KindlingProject/kindling) Under the Apache License 2.0
- [pyroscope](https://github.com/grafana/pyroscope) Under the Apache License 3.0
- [libbpfgo](https://github.com/aquasecurity/libbpfgo) Under the Apache License 2.0
- [bcc](https://github.com/iovisor/bcc) Under the Apache License 2.0
* ...
