package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.1.1"

// Format represents an enumeration of possible file path formats.
type Format int

const (
	// Windows file paths may contain a volume prefix as either a drive
	// letter "C:" or a UNC path "\\host\share". Following a drive letter,
	// an absolute or relative path may be specified, where the former is
	// expressed with a leading "\", which indicates the path is anchored to
	// the root of the volume. Relative paths are anchored to the current
	// directory. UNC paths must always be fully-qualified, absolute file
	// paths. For more info, see:
	//
	//   https://docs.microsoft.com/en-us/dotnet/standard/io/file-path-formats
	//
	// Both types of volumes, drive letter or UNC path, are mounted within
	// the WSL user space as any ordinary mount point, and therefore can be
	// converted to traditional Unix file paths. Both types of volumes are
	// mounted in WSL using the "drvfs" file system driver.
	//
	// The Windows directory separator is always "\".
	Windows Format = iota

	// Unix file paths do not distinguish the volume from which a given
	// file system is provided. Instead, volumes are mounted at any regular
	// directory path within a single/common file system. This file system
	// uses "/" as its root path. Any file path containing a leading "/"
	// is interpreted as an absolute file path; all others are considered
	// relative file paths.
	//
	// Currently, the WSL file system is stored on top of the Windows host
	// (NTFS) file system. However, accessing files stored in the WSL file
	// system from the Windows host context is not supported and may lead to
	// file system corruption. Read operations are considered relatively
	// safe, but write operations are not. The user must ensure any Windows
	// application used to read files from the WSL container does not also
	// write to that file system (e.g., cache or temporary files).
	//
	// The Unix directory separator is always "/".
	Unix

	// Any file paths are simple file names that do not contain a directory
	// separator, which are valid on both Windows and Unix file systems.
	Any
)

const (
	// NixPathEnvSuffix defines the suffix of WSL environment variable
	// identifiers for variables holding paths to Windows volumes mounted
	// within WSL user space. The prefix of these identifiers is constructed
	// dynamically based on the volume specified by a Windows absolute path.
	NixPathEnvSuffix = "_VOLUME_PATH" // e.g., C_VOLUME_PATH="/mnt/c"
	// WslRootfsEnvVar holds the Windows-formatted absolute path to the
	// active WSL distribution's rootfs directory (content is read-only).
	// For files that exist solely in the WSL virtual rootfs (they are not
	// stored physically on any host file system), the following environment
	// variable is used as the path prefix.
	WslRootfsEnvVar = "WSL_ROOTFS_PATH"
	UncPathEnvVar   = "WSL_UNC_PATH"
)

const (
	toWinFlagDesc = "Convert Unix to Windows file path(s)"
	toNixFlagDesc = "Convert Windows to Unix file path(s)"
	existFlagDesc = "Do not translate paths found only in WSL rootfs"
	svNumFlagDesc = "Print version number and exit"
)

func Usage() {
	for _, s := range []string{
		filepath.Base(os.Args[0]) + " version " + version,
		"",
		"Usage:",
		"\t" + os.Args[0] + " [-w|-x] [-e] [PATH ...]",
		"",
		"Options:",
		"\t-w    " + toWinFlagDesc,
		"\t-x    " + toNixFlagDesc,
		"\t-e    " + existFlagDesc,
		"\t-v    " + svNumFlagDesc,
		"",
		"\tIf no option specifying the target file path(s) format is given,",
		"\tthen the format is automatically determined by analyzing each given",
		"\tpath individually and using the opposite format(s), respectively.",
		"",
		"Environment:",
		"\tTranslating absolute file paths from one filesystem to the other",
		"\trequires the definition of environment variable(s) associating",
		"\tWindows volumes with WSL mount points.",
		"",
		"\tThese environment variables are named according to their Windows",
		"\tvolume name in all uppercase, appended with \"" + NixPathEnvSuffix + "\".",
		"\tFor example, converting \"C:\\Windows\" will look for an environment",
		"\tvariable such as: C" + NixPathEnvSuffix + "=\"/mnt/c\".",
		"",
		"\tIf a UNC path is provided, a special environment variable named",
		"\tWSL_UNC_PATH is read containing a list of all UNC path to mount",
		"\tpoint mappings, with the following semicolon-delimited format:",
		"",
		"\t    WSL_UNC_PATH='\\h1\\v1\\rp1=/lp1;\\h2\\v2\\rp2=/lp2'",
		"",
		"\tThese same rules are applied in reverse when converting Unix file",
		"\tpaths to Windows as well. The user's environment is inspected for",
		"\tall variables with the mentioned suffix and using whichever matches",
		"\tthe longest substring of the given path.",
		"",
		"\tIf the given Unix file path does not exist on any Windows file",
		"\tsystem (the above search will fail to find a corresponding key in",
		"\tthe user's environment), then the path is assumed to exist only on",
		"\tthe virtual Linux file system. In this case, a special environment",
		"\tvariable named " + WslRootfsEnvVar + " is consulted to resolve the Windows",
		"\tabsolute file path by appending the absolute Unix file path to the",
		"\tvalue of this environment variable. If the command-line flag -e is",
		"\tprovided, then this fallback is not performed, and any paths given",
		"\tthat do not have a corresponding mapping in the environment will",
		"\treturn an error.",
		"",
		"WARNING:",
		"\tWSL does not currently support writing to virtual Linux file",
		"\tsystems from a Windows context. Therefore, any paths resolved",
		"\tusing the path referenced in the " + WslRootfsEnvVar + " environment",
		"\tvariable should only be used for read-only operations. Writing",
		"\tto these paths could potentially corrupt a WSL file system!",
		"",
	} {
		fmt.Println("\t" + s)
	}
}

func main() {

	var (
		toWinFlag, toNixFlag, existFlag, svNumFlag bool
	)
	flag.BoolVar(&toWinFlag, "w", false, toWinFlagDesc)
	flag.BoolVar(&toNixFlag, "x", false, toNixFlagDesc)
	flag.BoolVar(&existFlag, "e", false, existFlagDesc)
	flag.BoolVar(&svNumFlag, "v", false, svNumFlagDesc)

	flag.Usage = Usage
	flag.Parse()

	if svNumFlag {
		fmt.Println(filepath.Base(os.Args[0]), "version", version)
	}

	if toWinFlag && toNixFlag {
		fmt.Fprintln(os.Stderr, "error: invalid arguments: -w and -x are mutually exclusive")
		os.Exit(100)
	}

	exitCode := 0

	// read from command line args if provided, otherwise STDIN
	s := bufio.NewScanner(InputReader(flag.Args()...))
	for s.Scan() {

		var err error
		text := s.Text()
		form := ""

		// use command line flag as target format if provided
		switch {
		case toWinFlag:
			form, _, err = Unix.Format(Windows, text, existFlag, 0)
		case toNixFlag:
			form, _, err = Windows.Format(Unix, text, existFlag, 0)
		default:
			// otherwise, no command line flag, try to detect the
			// given format and use the opposite as target format
			switch Identify(text) {
			case Windows:
				form, _, err = Windows.Format(Unix, text, existFlag, 0)
			case Unix:
				form, _, err = Unix.Format(Windows, text, existFlag, 0)
			case Any:
				form = Any.Clean(text)
			}
		}
		if nil != err {
			fmt.Fprintln(os.Stderr, "error: Format():", err)
			exitCode = 1
			continue
		}
		fmt.Println(form)
	}

	if err := s.Err(); nil != err {
		fmt.Fprintln(os.Stderr, "error: Scan():", err)
		os.Exit(127)
	}

	os.Exit(exitCode)
}

// InputReader returns an io.Reader that reads all given arguments if provided,
// otherwise it reads from STDIN.
func InputReader(args ...string) io.Reader {
	if len(args) > 0 {
		return strings.NewReader(strings.Join(args, "\n"))
	}
	return os.Stdin
}

// Identify automatically detects and returns the file path Format of a given
// string, by scanning for the first directory path separator.
// If no separator exists, such as a simple file name, then the path is
// checked if it contains a Windows drive letter prefix (a volume-anchored
// relative path, e.g., "D:foo.dat" refers to "foo.dat" in the current
// working path on the "D:" volume, regardless of the current volume).
// Otherwise, the path is valid for both systems, and the special Format
// value Any is returned.
func Identify(s string) Format {
	for _, c := range s {
		if c == '\\' {
			return Windows
		}
		if c == '/' {
			return Unix
		}
	}
	// No directory separator, check if it contains a drive letter prefix
	if len(s) > 1 {
		if d := s[0]; (s[1] == ':') &&
			(('a' <= d && d <= 'z') || ('A' <= d && d <= 'Z')) {
			return Windows
		}
	}
	return Any
}

// SplitVolume separates the given file path in Windows Format into volume and
// path components. Volume may be either a drive letter or a UNC host+share
// expression. If a volume expression does not exist, or Format is not Windows,
// then the returned volume is the empty string and path is unchanged.
func (f Format) SplitVolume(s string) (volume, path string) {

	// Windows is the only Format that uses volume prefixes
	if Windows != f {
		return "", s
	}
	// test if we have a drive letter X: prefix
	if len(s) < 2 {
		return "", s
	}
	if d := s[0]; (s[1] == ':') &&
		(('a' <= d && d <= 'z') || ('A' <= d && d <= 'Z')) {
		return s[:2], s[2:]
	}

	// test if we have a UNC \\host\share prefix
	if len(s) < 5 {
		return "", s
	}
	// verify we have leading slashes
	if s[:2] == `\\` && s[2] != '\\' && s[2] != '.' {
		for n := 3; n < len(s)-1; n++ {
			// walk over server name until we reach volume separator
			if s[n] == '\\' {
				n++
				if s[n] != '\\' {
					if s[n] == '.' {
						break
					}
					// we are in volume name,
					//   take remaining chars up to EOS or next separator
					for ; n < len(s); n++ {
						if s[n] == '\\' {
							break
						}
					}
					return s[:n], s[n:]
				}
				break

			}
		}
	}
	return "", s
}

// Elements splits the given file path into individual path components based
// on the receiver Format f's directory separator. Unlike strings.Split, empty
// components are not added to the returned slice.
func (f Format) Elements(s string) []string {
	e := []string{}
	b := strings.Builder{}
	for _, c := range s {
		if f.issep(c) {
			e = append(e, b.String())
			b.Reset()
		} else {
			b.WriteRune(c)
		}
	}
	if b.Len() > 0 {
		e = append(e, b.String())
	}
	return e
}

// Clean is the same as standard Go's path/filepath.Clean, except that it can
// handle arbitrary directory separators. In particular, it applies the
// following rules iteratively until no further processing can be done:
//
//     1. Replace multiple directory separators with a single one.
//     2. Eliminate each "." path name element (the current directory).
//     3. Eliminate each inner-".." path name element (the parent directory)
//        along with the non-".." element that precedes it.
//     4. Eliminate ".." elements that begin a rooted path: that is, replace
//        "/.." by "/" at the beginning of a path, assuming directory separator
//        is '/'.
//
// The volume prefix, if provided as either a drive letter or UNC host+share, is
// preserved on both absolute and relative file paths.
//
// The returned path ends in a slash only if it represents a root directory,
// such as "/" on Unix or `C:\` on Windows.
//
// If the result of this process is an empty string, "." is returned.
func (f Format) Clean(s string) string {

	var vol string
	vol, s = f.SplitVolume(s)

	if len(s) == 0 {
		return vol + "."
	}

	// replace multiple separator elements with a single one
	u := string(f.sep())
	d := u + u
	for n := 0; n != len(s); {
		n = len(s)
		s = strings.ReplaceAll(s, d, u)
	}

	e := f.Elements(s)

	// remove any "." elements (current dir)
	p := []string{}
	for _, u := range e {
		if u != "." {
			p = append(p, u)
		}
	}

	// remove any (inner) ".." elements and their predecessor (parent dir)
	for {
		// create buffer for current pass
		q := []string{}
		// keep iterating until no change was performed (done == true)
		done, skip := true, false
		// walk over each path element, checking if its following
		// element is ".."
		for i := range p {
			if !skip {
				// preceding element was not ".."
				if (i+1 < len(p)) && (p[i] != "..") && (p[i+1] == "..") {
					if (i == 0) && (p[i] == "") {
						// keep. leading element is ".."
						q = append(q, p[i])
					}
					// skip. following element is ".."
					// need to process elements again
					done, skip = false, true
				} else {
					// keep. following element is not ".."
					q = append(q, p[i])
				}
			} else {
				// skip. current element is ".."
				skip = false
			}
		}
		// replace final path elements with result of current pass
		p = q
		if done {
			// no change performed in current pass. all done.
			break
		}
	}

	// if no elements remain, use current dir "."
	if len(p) == 0 {
		return vol + "."
	} else {
		if (len(p) == 1) && (p[0] == "") {
			return vol + string(f.sep())
		} else {
			return vol + strings.Join(p, string(f.sep()))
		}
	}
}

// Format translates the given file path s, interpreted as a path in the
// receiver Format f, to a file path in given Format t.
//
// When translating absolute paths from one file system to the other,
// environment variables are used to determine relative paths or mount points.
//
// The bool return paramter is true if and only if the returned path is
// a Windows formatted path into the WSL virtual rootfs (i.e., read-only).
func (f Format) Format(t Format, s string, x bool, z uint) (string, bool, error) {

	s = f.Clean(s)
	wsl := false

	if z > 1 {
		return "", false, fmt.Errorf("invalid path: %s", s)
	}

	switch f {
	case Windows:
		if Unix == t {
			v, p := f.SplitVolume(s)
			if len(v) >= 2 {
				// absolute path
				v0, v1 := v[0], v[1]
				if (v1 == ':') && (('a' <= v0 && v0 <= 'z') || ('A' <= v0 && v0 <= 'Z')) {
					// convert drive letter to environment variable
					e := strings.ToUpper(string(v0)) + NixPathEnvSuffix
					if dp, ok := os.LookupEnv(e); ok {
						// replace drive letter with value of environment variable
						s = dp + p
					} else {
						return "", false, fmt.Errorf("environment variable not set: %s", e)
					}
				} else if len(v) >= 5 {
					v2 := v[2]
					if v[:2] == `\\` && v2 != '\\' && v2 != '.' {
						if up, ok := os.LookupEnv(UncPathEnvVar); ok {
							ok = false
							for _, vm := range strings.Split(up, `;`) {
								if m := strings.SplitN(vm, `=`, 2); len(m) == 2 {
									if strings.ToUpper(m[0]) == strings.ToUpper(v) {
										s = m[1] + p
										ok = true
										break
									}
								}
							}
							if !ok {
								return "", false, fmt.Errorf("UNC volume %q not found in environment variable: %s=%q", v, UncPathEnvVar, up)
							}
						} else {
							return "", false, fmt.Errorf("environment variable not set: %s", UncPathEnvVar)
						}
					}
				}
			}
			s = strings.ReplaceAll(s, string(f.sep()), string(t.sep()))
			s = t.Clean(s)
		}

	case Unix:
		if Windows == t {
			if len(s) > 0 {
				if s[0] == '/' {
					// absolute file path
					//e, err := filepath.EvalSymlinks(s)
					//if err != nil {
					//	return "", false, err
					//}
					s = f.abspath(s)
					var rk, rv string
					unc := false
					if up, ok := os.LookupEnv(UncPathEnvVar); ok {
						for _, vm := range strings.Split(up, `;`) {
							if m := strings.SplitN(vm, `=`, 2); len(m) == 2 {
								if strings.HasPrefix(s, m[1]) && len(m[1]) > len(rv) {
									rk, rv = m[0], m[1]
									unc = true
								}
							}
						}
					}
					if unc {
						s = strings.Replace(s, rv, rk, 1)
					} else {
						for _, e := range os.Environ() {
							n := strings.IndexRune(e, '=')
							if (-1 != n) && (len(e) > n+1) {
								k, v := e[:n], f.Clean(e[n+1:])
								if strings.HasSuffix(k, NixPathEnvSuffix) &&
									strings.HasPrefix(s, v) && (len(v) > len(rv)) {
									rk, rv = k, v
								}
							}
						}
						if len(rk) > 0 {
							s = strings.Replace(s, rv, string(rk[0])+":", 1)
						} else {
							if up, ok := os.LookupEnv(WslRootfsEnvVar); !x && ok {
								s = fmt.Sprintf("%s%c%s", up, t.sep(), s)
								wsl = true
							} else {
								return "", false, fmt.Errorf("path substring not found in environment: %s", s)
							}
						}
					}
				} else {
					// relative file path
					//a, err := filepath.Abs(s)
					//if err != nil {
					//	return "", false, err
					//}
					// if we cannot resolve the absolute path to a Windows volume, then
					// the relative path will never make sense in a Windows context.
					// Instead, construct an absolute path to the WSL rootfs path.
					p, w, err := f.Format(t, f.abspath(s), x, z+1)
					if err != nil {
						return "", false, err
					}
					if w {
						// The absolute path was unresolved to a Windows volume, and we
						// instead received a path to the virtual WSL rootfs.
						// Use the absolute WSL rootfs path instead of a relative path.
						s, wsl = p, true
					}
					// s is either a physical relative path or an absolute virtual path.
				}
			}

			s = strings.ReplaceAll(s, string(f.sep()), string(t.sep()))
			s = t.Clean(s)
		}

	case Any:
	default:
	}

	return s, wsl, nil
}

func (f Format) abspath(s string) string {
	var act, rel string
	for _, p := range strings.Split(s, string(f.sep())) {
		if act == "" && rel == "" && p == "" {
			p = string(f.sep())
		} else {
			if rel != "" {
				if rel != string(f.sep()) {
					rel += string(f.sep())
				}
				rel += p
				continue
			}
		}
		t := act
		if t != "" && t != string(f.sep()) {
			t += string(f.sep())
		}
		t += p
		es, ee := filepath.EvalSymlinks(t)
		as, ae := filepath.Abs(es)
		if ee == nil && ae == nil {
			act = as
		} else {
			rel = p
		}
	}
	if act != "" {
		if rel != "" {
			return act + string(f.sep()) + rel
		}
		return act
	}
	if rel != "" {
		return rel
	}
	return ""
}

// issep returns true if and only if the given rune is equal to the receiver
// Format f's directory separator.
func (f Format) issep(c rune) bool {
	switch f {
	case Windows:
		return c == '\\'
	case Unix:
		return c == '/'
	case Any:
		return c == '/' || c == '\\'
	}
	return false
}

// sep returns the directory separator rune of the receiver Format f.
func (f Format) sep() rune {
	if Windows == f {
		return '\\'
	}
	return '/'
}
