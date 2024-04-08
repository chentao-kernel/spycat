# spycat
An eBPF observable agent to solve os performance issue in kernel or app
## Architecture
<div align=center> <img src="doc/spycat.png" width = "60%" height="60%" /> </div>

## Build & Run
### Build
You can use docker/build.sh to build in a container.
```
usage: ./args.sh [-h|--help -b|--build -c|--compile -t|--tar -V|--bin_ver]
        -b              build image
        -c              compile
        -t              tar binary
        -V 0.1.2 -t     tar binary with 0.1.2 version
```
### Run
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
## License
spycat is distributed under [Apache License, Version2.0]
## Thanks
* Kindling Pyroscope libbpf bcc
* ...
