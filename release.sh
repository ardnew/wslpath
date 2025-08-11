#!/bin/bash

v=$( command grep -oP 'const version = "\K[^"]+' main.go )

r="release/wslpath$v"

# files to package with the release executable
f=( LICENSE README.md )

if ! go build ; then
  echo "error: failed to build"
  exit 1
fi

update-readme() {

  cat <<__usage__
## Usage

Use the \`-h\` flag for details:

\`\`\`
$( ./wslpath -h | tr -d '\r' )
\`\`\`

__usage__

  cat <<__install__
## Installation

### Install pre-compiled release packages

Download latest release package for your system[^1], extract contents, and copy the \`wslpath\` executable somewhere in your \`\$PATH\`.

[^1]: Use one of the Linux release packages if you only intend to use \`wslpath\` from a WSL environment.

__install__

  printf '|`GOOS`   |`GOARCH`|.zip|.tar.gz|.tar.bz2|\n'
  printf '|:-------:|:------:|:--:|:-----:|:------:|\n'

  for p in {linux,windows}-{386,amd64,arm,arm64} ; do 

    x=( $( tr '-' '\n' <<< $p ) )
    o=${x[0]}
    a=${x[1]}

    u="https://github.com/ardnew/wslpath/releases/v${v}/${r##*/}.${o}-${a}"

    printf '|%-9s|%-8s|%s|%s|%s|\n' '`'"${o}"'`' '`'"${a}"'`' \
      "[:floppy_disk:](${u}.zip)" \
      "[:floppy_disk:](${u}.tar.gz)" \
      "[:floppy_disk:](${u}.tar.bz2)"

  done

  cat <<__compile__

### Compile and install with local source code

\`\`\`sh
git clone https://github.com/ardnew/wslpath
cd wslpath
go install -v
\`\`\`

### Compile and install using module-aware toolchain (Go 1.16 or later)

\`\`\`sh
go install -v github.com/ardnew/wslpath@latest
\`\`\`

###### Compile and install using legacy GOPATH toolchain (prior Go 1.16)

\`\`\`sh
GO111MODULE=off go get -v github.com/ardnew/wslpath
\`\`\`

__compile__

}

readme=$( mktemp --tmpdir )
trap "rm -fv '${readme}'" ERR EXIT

perl -pe 'if (/^## Usage/) { $_ = undef; last }' < README.md > "${readme}"
update-readme >> "${readme}"
mv "${readme}" README.md

for p in {linux,windows}-{386,amd64,arm,arm64} ; do 

  x=( $( tr '-' '\n' <<< "${p}" ) )
  o=${x[0]}
  a=${x[1]}

  echo
  echo "==== ${o}-${a} ===="
  echo

  mkdir -p "${r}"
  GOOS="${o}" GOARCH="${a}" go build -o "${r}"
  cp -v "${f[@]}" "${r}"

  pushd "${r%/*}" &>/dev/null
  zip -vr "wslpath${v}.${o}-${a}.zip" "${r##*/}"
  tar -czvf "wslpath${v}.$o-${a}.tar.gz" "${r##*/}"
  tar -cjvf "wslpath${v}.${o}-${a}.tar.bz2" "${r##*/}"
  popd &>/dev/null

  rm -rf "${r}" 

done

cmd=( gh release create v${v} --generate-notes release/wslpath${v}.* )

echo
echo "publish release:"
echo "  ${cmd[@]}"
echo
read -r -e -N 1 -p "create? [yN] "
[[ "${REPLY}" == [yY] ]] || exit
exec "${cmd[@]}"
