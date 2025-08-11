[docimg]:https://godoc.org/github.com/ardnew/wslpath?status.svg
[docurl]:https://godoc.org/github.com/ardnew/wslpath
[repimg]:https://goreportcard.com/badge/github.com/ardnew/wslpath
[repurl]:https://goreportcard.com/report/github.com/ardnew/wslpath

# wslpath
#### Convert between Windows and Linux file paths in WSL

[![GoDoc][docimg]][docurl] [![Go Report Card][repimg]][repurl]

## Configuration

```sh
# define your environment's mount points
export C_VOLUME_PATH='/mnt/c'
export D_VOLUME_PATH='/mnt/backup'

# define any mounted network shares with UNC paths
export WSL_UNC_PATH='\\work\s1\path=/mnt/work/s1;\\work\s2\path=/mnt/work/s2'  

# define your rootfs path for any files that reside only in WSL (see following section)
export WSL_ROOTFS_PATH=$( wslrootfs )
```

#### Determine active WSL distribution and rootfs path

The following shell functions together will return your active WSL distro and its corresponding rootfs Windows path. The latter can be used to initialize the `WSL_ROOTFS_PATH` environment variable used by `wslpath` for files that reside only in your WSL virtual file system.

```bash
wsldistro() {
        reg.exe QUERY 'HKCU\Software\Microsoft\Windows\CurrentVersion\Lxss' /v DefaultDistribution /t REG_SZ | 
                command grep -oP 'DefaultDistribution\s+REG_SZ\s+\K{[^\}]+}'
}

wslrootfs() {
        reg.exe QUERY 'HKCU\Software\Microsoft\Windows\CurrentVersion\Lxss\'"$(wsldistro)" /v BasePath /t REG_SZ | 
                command grep -oP 'BasePath\s+REG_SZ\s+\K\S.+' |
                sed -E 's|/*\s*$|\\rootfs|'
}
```

## Examples (see [Configuration](README.md#Configuration) section for reference)

```bash
# Linux paths do a reverse lookup on the environment:
$ wslpath /mnt/d/andrew/file         # -> D:\andrew\file

# you'll need to use single quotes or double \\ for Windows paths:
$ wslpath 'C:\Windows\System32'      # -> /mnt/c/Windows/System32

# UNC paths are also resolved:
$ wslpath '\\work\s1\path\foo'       # -> /mnt/work/s1/foo
$ wslpath '/mnt/work/s2/foo/bar'     # -> \\work\s2\path\foo\bar

# some paths do not exist outside of WSL:
$ wslpath '/etc'                     # -> C:\Users\andrew\AppData\Local\Packages\CanonicalGroupLimited.UbuntuonWindows_79rhkp1fndgsc\LocalState\rootfs\etc

# it will also read from stdin
$ echo /mnt/c/Windows | wslpath      # -> C:\Windows
```

### Integration

The following shell functions are convenient `wslpath` wrappers for working with Windows:

```bash
# convert all given paths to absolute and translate to Windows file paths.
# if no arguments are given, use $PWD.
winpath() {
        [[ ${#} -gt 0 ]] || set -- "${PWD}"
        realpath -eq "${@}" | wslpath -w
}

# use Windows Explorer to open all given files, so you can easily open documents 
# and folders directly from the WSL command line with Unix file path arguments.
open() {
        while read -re path; do
                explorer.exe "${path}"
        done < <( winpath "${@}" )
 }
```

#### Integration Examples

```sh
# Print Windows file paths from relative Unix paths
$ cd /mnt/c/Users/andrew/Documents
$ winpath                             # -> C:\Users\andrew\Documents
$ winpath ..                          # -> C:\Users\andrew

# Open the current working directory in Windows Explorer
$ open                                # -> explorer.exe C:\Users\andrew\Documents
                                      
# Use MS Word to open .docx files and Adobe Acrobat to read a PDF
$ open Research.docx Reference.pdf    # -> explorer.exe C:\Users\andrew\Documents\Research.docx
                                      # -> explorer.exe C:\Users\andrew\Documents\Reference.pdf
```

## Usage

Use the `-h` flag for details:

```
	wslpath version 0.2.0
	
	Usage:
		./wslpath [-a] [-w|-x] [-e] [PATH ...]
	
	Options:
		-a    Output converted file paths as absolute file path
		-w    Convert Unix to Windows file path(s)
		-x    Convert Windows to Unix file path(s)
		-e    Do not translate paths found only in WSL rootfs
		-v    Print version number and exit
	
		If no option specifying the target file path(s) format is given,
		then the format is automatically determined by analyzing each given
		path individually and using the opposite format(s), respectively.
	
	Environment:
		Translating absolute file paths from one filesystem to the other
		requires the definition of environment variable(s) associating
		Windows volumes with WSL mount points.
	
		These environment variables are named according to their Windows
		volume name in all uppercase, appended with "_VOLUME_PATH".
		For example, converting "C:\Windows" will look for an environment
		variable such as: C_VOLUME_PATH="/mnt/c".
	
		If a UNC path is provided, a special environment variable named
		WSL_UNC_PATH is read containing a list of all UNC path to mount
		point mappings, with the following semicolon-delimited format:
	
		    WSL_UNC_PATH='\h1\v1\rp1=/lp1;\h2\v2\rp2=/lp2'
	
		These same rules are applied in reverse when converting Unix file
		paths to Windows as well. The user's environment is inspected for
		all variables with the mentioned suffix and using whichever matches
		the longest substring of the given path.
	
		If the given Unix file path does not exist on any Windows file
		system (the above search will fail to find a corresponding key in
		the user's environment), then the path is assumed to exist only on
		the virtual Linux file system. In this case, a special environment
		variable named WSL_ROOTFS_PATH is consulted to resolve the Windows
		absolute file path by appending the absolute Unix file path to the
		value of this environment variable. If the command-line flag -e is
		provided, then this fallback is not performed, and any paths given
		that do not have a corresponding mapping in the environment will
		return an error.
	
	WARNING:
		WSL does not currently support writing to virtual Linux file
		systems from a Windows context. Therefore, any paths resolved
		using the path referenced in the WSL_ROOTFS_PATH environment
		variable should only be used for read-only operations. Writing
		to these paths could potentially corrupt a WSL file system!
	
```

## Installation

### Install pre-compiled release packages

Download latest release package for your system[^1], extract contents, and copy the `wslpath` executable somewhere in your `$PATH`.

[^1]: Use one of the Linux release packages if you only intend to use `wslpath` from a WSL environment.

|`GOOS`   |`GOARCH`|.zip|.tar.gz|.tar.bz2|
|:-------:|:------:|:--:|:-----:|:------:|
|`linux`  |`386`   |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-386.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-386.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-386.tar.bz2)|
|`linux`  |`amd64` |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-amd64.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-amd64.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-amd64.tar.bz2)|
|`linux`  |`arm`   |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-arm.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-arm.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-arm.tar.bz2)|
|`linux`  |`arm64` |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-arm64.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-arm64.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.linux-arm64.tar.bz2)|
|`windows`|`386`   |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-386.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-386.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-386.tar.bz2)|
|`windows`|`amd64` |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-amd64.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-amd64.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-amd64.tar.bz2)|
|`windows`|`arm`   |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-arm.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-arm.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-arm.tar.bz2)|
|`windows`|`arm64` |[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-arm64.zip)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-arm64.tar.gz)|[:floppy_disk:](https://github.com/ardnew/wslpath/releases/0.2.0/wslpath0.2.0.windows-arm64.tar.bz2)|

### Compile and install with local source code

```sh
git clone https://github.com/ardnew/wslpath
cd wslpath
go install -v
```

### Compile and install using module-aware toolchain (Go 1.16 or later)

```sh
go install -v github.com/ardnew/wslpath@latest
```

###### Compile and install using legacy GOPATH toolchain (prior Go 1.16)

```sh
GO111MODULE=off go get -v github.com/ardnew/wslpath
```

