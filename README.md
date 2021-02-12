[docimg]:https://godoc.org/github.com/ardnew/wslpath?status.svg
[docurl]:https://godoc.org/github.com/ardnew/wslpath
[repimg]:https://goreportcard.com/badge/github.com/ardnew/wslpath
[repurl]:https://goreportcard.com/report/github.com/ardnew/wslpath

# wslpath
#### wslpath

[![GoDoc][docimg]][docurl] [![Go Report Card][repimg]][repurl]

## Usage

How to use:

```sh
# define your environment's mount points
export C_VOLUME_PATH=/mnt/c
# UNC paths also work
export MYSERVER__SHARE_VOLUME_PATH=/mnt/myserver/share
# you'll need to use single quotes or double \\ for Windows paths
wslpath 'C:\Windows\System32\..'              # -> /mnt/c/Windows
wslpath '\\myserver\share\blah'               # -> /mnt/myserver/share/blah
wslpath /mnt/c/andrew/file                    # -> C:\andrew\file

```

Use the `-h` flag for usage summary:

```
        Usage:
                wslpath [-w|-x] [PATH ...]

        Options:
                --windows, -w        Convert Unix to Windows file path(s)
                --unix, -x           Convert Windows to Unix file path(s)

        Environment:
                Translating absolute file paths from one filesystem to the other
                requires the definition of environment variable(s) associating
                Windows volumes with WSL mount points.

                These environment variables are named according to their Windows
                volume name in all uppercase, appended with "_VOLUME_PATH".
                For example, converting "C:\Windows" will look for an environment
                variable such as: C_VOLUME_PATH="/mnt/c".

                If a UNC path is provided, the environment variable identifier
                follows a similar convention, all uppercase, and includes both the
                host and share componenets separated by two underscores, with any
                period replaced by a lower-case 'p', and all other remaining
                non-alphanumeric characters replaced with a single underscore.
                For example, "\\dev_okc.net\aps\share" would look for a
                variable named DEV_OKCpNET__APS_VOLUME_PATH.

                These same rules are applied in reverse when converting Unix file
                paths to Windows as well. The user's environment is inspected for
                all variables with the mentioned suffix and using whichever matches
                the longest substring of the given path.
```

## Installation

Use the builtin Go package manager:

```sh
go get -v github.com/ardnew/wslpath
```
