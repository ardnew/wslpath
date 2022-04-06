#!/bin/bash

v=$( command grep -oP 'const version = "\K[^"]+' main.go )

r="release/wslpath$v"

# files to package with the release executable
f=( LICENSE README.md )

for p in {linux,freebsd,windows}-{386,amd64,arm,arm64} darwin-{amd64,arm64} ; do 

	x=( $( tr '-' '\n' <<< $p ) )
	o=${x[0]}
	a=${x[1]}

	echo
	echo "==== $o-$a ===="
	echo

	mkdir -p "$r"
	GOOS="$o" GOARCH="$a" go build -o "$r"
	cp -v "${f[@]}" "$r"

	pushd "${r%/*}" &>/dev/null
	zip -vr "wslpath$v.$o-$a.zip" "${r##*/}"
	tar -czvf "wslpath$v.$o-$a.tar.gz" "${r##*/}"
	tar -cjvf "wslpath$v.$o-$a.tar.bz2" "${r##*/}"
	popd &>/dev/null

	rm -rf "$r" 

done

# table to copy into README.md release package list
for p in {linux,freebsd,windows}-{386,amd64,arm,arm64} darwin-{amd64,arm64} ; do 

	x=( $( tr '-' '\n' <<< $p ) )
	o=${x[0]}
	a=${x[1]}

	u="https://github.com/ardnew/wslpath/releases/$v/${r##*/}.$o-$a"

	printf '|%-9s|%-8s|%s|%s|%s|\n' '`'"$o"'`' '`'"$a"'`' \
		"[:floppy_disk:]($u.zip)" \
		"[:floppy_disk:]($u.tar.gz)" \
		"[:floppy_disk:]($u.tar.bz2)"

done
