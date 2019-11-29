// Package file contains functions for low-level migration files handling.
package file

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/db-journey/migrate/v2/direction"
)

var filenameRegex = `^([0-9]+)_(.*)\.(up|down)\.%s(?:\.tpl)?$`

// FilenameRegex builds regular expression stmt with given
// filename extension from driver.
func FilenameRegex(filenameExtension string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(filenameRegex, filenameExtension))
}

type Version uint64     // Version is the migration version.
type Versions []Version // Versions is the list of migrations.

// Contains checks if a _version_ is contained in the list of migrations.
func (versions Versions) Contains(version Version) bool {
	for _, v := range versions {
		if v == version {
			return true
		}
	}
	return false
}

// Len is the number of elements in the collection.
// Required by Sort Interface{}
func (versions Versions) Len() int {
	return len(versions)
}

// Less reports whether the element with index i should sort before the element
// with index j. Required by Sort Interface{}.
func (versions Versions) Less(i, j int) bool {
	return versions[i] < versions[j]
}

// Swap swaps the elements with indexes i and j. Required by Sort Interface{}.
func (versions Versions) Swap(i, j int) {
	versions[i], versions[j] = versions[j], versions[i]
}

// File represents one file on disk.
// Example: 20060102150405_initial_plan_to_do_sth.up.sql
type File struct {
	// absolute path to file
	Path string

	// the name of the file
	FileName string

	// version parsed from filename
	Version Version

	// the actual migration name parsed from filename
	Name string

	// content of the file
	Content []byte

	// UP or DOWN migration
	Direction direction.Direction
}

// Files is a slice of Files.
type Files []File

// MigrationFile represents both the UP and the DOWN migration file.
type MigrationFile struct {
	// version of the migration file, parsed from the filenames
	Version Version

	// reference to the *up* migration file
	UpFile *File

	// reference to the *down* migration file
	DownFile *File
}

// MigrationFiles is a slice of MigrationFiles.
type MigrationFiles []MigrationFile

// ReadContent reads the file into the content if it's empty.
func (f *File) ReadContent() error {
	if len(f.Content) == 0 {
		content, err := ioutil.ReadFile(path.Join(f.Path, f.FileName))
		if err != nil {
			return err
		}

		// if the file is a template (extension = ".tpl"), it will be parsed and executed with the program current env
		if f.IsTemplate() {
			tmpl, err := template.New(f.Name).Parse(string(content[:]))
			if err != nil {
				return err
			}
			getenvironment := func(data []string, getkeyval func(item string) (key, val string)) map[string]string {
				items := make(map[string]string)
				for _, item := range data {
					key, val := getkeyval(item)
					items[key] = val
				}
				return items
			}
			environment := getenvironment(os.Environ(), func(item string) (key, val string) {
				splits := strings.Split(item, "=")
				key = splits[0]
				val = splits[1]
				return
			})

			var b bytes.Buffer
			if err := tmpl.ExecuteTemplate(&b, f.Name, environment); err != nil {
				return err
			}
			content = b.Bytes()
		}

		f.Content = content
	}
	return nil
}

// IsTemplate returns true if the current file is a Template
func (f *File) IsTemplate() bool {
	if filepath.Ext(f.FileName) == ".tpl" {
		return true
	}
	return false
}

// Pending returns the list of pending migration files.
func (mf *MigrationFiles) Pending(versions Versions) (Files, error) {
	sort.Sort(mf)
	files := make(Files, 0)
	for _, migrationFile := range *mf {
		if !versions.Contains(migrationFile.Version) && migrationFile.UpFile != nil {
			files = append(files, *migrationFile.UpFile)
		}
	}
	return files, nil
}

// Applied returns the list of applied migration files.
func (mf *MigrationFiles) Applied(versions Versions) (Files, error) {
	sort.Sort(sort.Reverse(mf))
	files := make(Files, 0)
	for _, migrationFile := range *mf {
		if versions.Contains(migrationFile.Version) && migrationFile.DownFile != nil {
			files = append(files, *migrationFile.DownFile)
		}
	}
	return files, nil
}

// Relative travels relatively through migration files.
//
// 		+1 will fetch the next up migration file
// 		+2 will fetch the next two up migration files
// 		+n will fetch ...
// 		-1 will fetch the the previous down migration file
// 		-2 will fetch the next two previous down migration files
//		-n will fetch ...
func (mf *MigrationFiles) Relative(relativeN int, versions Versions) (Files, error) {
	var d direction.Direction
	if relativeN > 0 {
		d = direction.Up
	} else if relativeN < 0 {
		d = direction.Down
	} else { // relativeN == 0
		return nil, nil
	}

	var files Files
	var err error
	if d == direction.Up {
		files, err = mf.Pending(versions)
	} else {
		relativeN = -relativeN
		files, err = mf.Applied(versions)
	}
	if relativeN > len(files) {
		relativeN = len(files)
	}
	return files[:relativeN], err
}

// ReadMigrationFiles reads all migration files from a given path.
func ReadMigrationFiles(path string, filenameRegex *regexp.Regexp) (files MigrationFiles, err error) {
	// find all migration files in path.
	ioFiles, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	type tmpFile struct {
		version  Version
		name     string
		filename string
		d        direction.Direction
	}
	var tmpFiles []*tmpFile
	tmpFileMap := map[Version]map[direction.Direction]tmpFile{}
	for _, file := range ioFiles {
		version, name, d, err := parseFilenameSchema(file.Name(), filenameRegex)
		if err == nil {
			if _, ok := tmpFileMap[version]; !ok {
				tmpFileMap[version] = map[direction.Direction]tmpFile{}
			}
			if existing, ok := tmpFileMap[version][d]; !ok {
				tmpFileMap[version][d] = tmpFile{version: version, name: name, filename: file.Name(), d: d}
			} else {
				return nil, fmt.Errorf("duplicate migration file version %d : %q and %q", version, existing.filename, file.Name())
			}
			tmpFiles = append(tmpFiles, &tmpFile{version, name, file.Name(), d})
		}
	}

	// put tmpFiles into MigrationFile struct.
	parsedVersions := make(map[Version]bool)
	newFiles := make(MigrationFiles, 0)
	for _, file := range tmpFiles {
		if _, ok := parsedVersions[file.version]; !ok {
			migrationFile := MigrationFile{
				Version: file.version,
			}

			var lookFordirection direction.Direction
			switch file.d {
			case direction.Up:
				migrationFile.UpFile = &File{
					Path:      path,
					FileName:  file.filename,
					Version:   file.version,
					Name:      file.name,
					Content:   nil,
					Direction: direction.Up,
				}
				lookFordirection = direction.Down
			case direction.Down:
				migrationFile.DownFile = &File{
					Path:      path,
					FileName:  file.filename,
					Version:   file.version,
					Name:      file.name,
					Content:   nil,
					Direction: direction.Down,
				}
				lookFordirection = direction.Up
			default:
				return nil, errors.New("Unsupported direction.Direction Type")
			}

			for _, file2 := range tmpFiles {
				if file2.version == file.version && file2.d == lookFordirection {
					switch lookFordirection {
					case direction.Up:
						migrationFile.UpFile = &File{
							Path:      path,
							FileName:  file2.filename,
							Version:   file.version,
							Name:      file2.name,
							Content:   nil,
							Direction: direction.Up,
						}
					case direction.Down:
						migrationFile.DownFile = &File{
							Path:      path,
							FileName:  file2.filename,
							Version:   file.version,
							Name:      file2.name,
							Content:   nil,
							Direction: direction.Down,
						}
					}
					break
				}
			}

			newFiles = append(newFiles, migrationFile)
			parsedVersions[file.version] = true
		}
	}

	sort.Sort(newFiles)
	return newFiles, nil
}

// parseFilenameSchema parses the filename.
func parseFilenameSchema(filename string, filenameRegex *regexp.Regexp) (version Version, name string, d direction.Direction, err error) {
	matches := filenameRegex.FindStringSubmatch(filename)
	if len(matches) != 4 {
		return 0, "", 0, errors.New("Unable to parse filename schema")
	}

	v, err := strconv.ParseUint(matches[1], 10, 0)
	version = Version(v)
	if err != nil {
		return 0, "", 0, errors.New(fmt.Sprintf("Unable to parse version '%v' in filename schema", matches[0]))
	}

	if matches[3] == "up" {
		d = direction.Up
	} else if matches[3] == "down" {
		d = direction.Down
	} else {
		return 0, "", 0, errors.New(fmt.Sprintf("Unable to parse up|down '%v' in filename schema", matches[3]))
	}

	return version, matches[2], d, nil
}

// Len is the number of elements in the collection. Required by Sort Interface{}.
func (mf MigrationFiles) Len() int {
	return len(mf)
}

// Less reports whether the element with index i should sort before the element
// with index j. Required by Sort Interface{}.
func (mf MigrationFiles) Less(i, j int) bool {
	return mf[i].Version < mf[j].Version
}

// Swap swaps the elements with indexes i and j. Required by Sort Interface{}.
func (mf MigrationFiles) Swap(i, j int) {
	mf[i], mf[j] = mf[j], mf[i]
}

// LineColumnFromOffset reads data and returns line and column integer for a given offset.
func LineColumnFromOffset(data []byte, offset int) (line, column int) {
	// TODO is there a better way?
	fs := token.NewFileSet()
	tf := fs.AddFile("", fs.Base(), len(data))
	tf.SetLinesForContent(data)
	pos := tf.Position(tf.Pos(offset))
	return pos.Line, pos.Column
}

// LinesBeforeAndAfter reads n lines before and after a given line.
// Set lineNumbers to true, to prepend line numbers.
func LinesBeforeAndAfter(data []byte, line, before, after int, lineNumbers bool) []byte {
	// TODO: Trim empty lines at the beginning and at the end
	// TODO: Trim offset whitespace at the beginning of each line, so that indentation is preserved
	startLine := line - before
	endLine := line + after
	lines := bytes.SplitN(data, []byte("\n"), endLine+1)

	if startLine < 0 {
		startLine = 0
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	selectLines := lines[startLine:endLine]
	newLines := make([][]byte, 0)
	lineCounter := startLine + 1
	lineNumberDigits := len(strconv.Itoa(len(selectLines)))
	for _, l := range selectLines {
		lineCounterStr := strconv.Itoa(lineCounter)
		if len(lineCounterStr)%lineNumberDigits != 0 {
			lineCounterStr = strings.Repeat(" ", lineNumberDigits-len(lineCounterStr)%lineNumberDigits) + lineCounterStr
		}

		lNew := l
		if lineNumbers {
			lNew = append([]byte(lineCounterStr+": "), lNew...)
		}
		newLines = append(newLines, lNew)
		lineCounter += 1
	}

	return bytes.Join(newLines, []byte("\n"))
}
