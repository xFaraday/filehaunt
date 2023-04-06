package filelib

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/klauspost/compress/zstd"
)

var (
	dirhome       = "/opt/filehaunt"
	dirforlogging = "/opt/filehaunt/logs"
	dirforbackups = "/opt/filehaunt/backups/"
)

type finfo struct {
	Name string
	Size int64
	Time string
	Hash string
}

func ContainsInt(s []int, e int) bool {
	sort.Ints(s)
	i := sort.SearchInts(s, e)
	return i < len(s) && s[i] == e
}

func CheckFile(name string) finfo {
	fileInfo, err := os.Stat(name)
	if err != nil {
		i := finfo{
			Name: "",
			Size: 0,
			Time: "",
			Hash: "",
		}
		return i
	}
	if fileInfo.IsDir() {

		t := fileInfo.ModTime().String()
		b := fileInfo.Size()

		i := finfo{
			Name: name,
			Size: b,
			Time: t,
			Hash: "directory",
		}

		return i
	} else {
		f, err := os.Open(name)
		if err != nil {
			panic(err)
		}
		if err != nil {
			if os.IsNotExist(err) {
				println("file not found:", fileInfo.Name())
			}
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			panic(err)
		}
		hash := h.Sum(nil)
		Enc := base64.StdEncoding.EncodeToString(hash)

		t := fileInfo.ModTime().String()
		b := fileInfo.Size()

		i := finfo{
			Name: name,
			Size: b,
			Time: t,
			Hash: Enc,
		}
		return i
	}
}

func Compress(in io.Reader, out io.Writer) error {
	enc, err := zstd.NewWriter(out)
	if err != nil {
		return err
	}
	//gets data from in and writes it to enc, which is out
	_, err = io.Copy(enc, in)
	if err != nil {
		enc.Close()
		return err
	}
	return enc.Close()
}

func Decompress(in io.Reader, out io.Writer) error {
	d, err := zstd.NewReader(in)
	if err != nil {
		return err
	}
	defer d.Close()

	// Copy content...
	_, err = io.Copy(out, d)
	return err
}

func OpenFile(file string) []string {
	var s []string
	stats := CheckFile(file)
	if stats.Size != 0 {
		f, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		// remember to close the file at the end of the program
		defer f.Close()

		// read the file line by line using scanner
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			// do something with a line
			s = append(s, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			panic(err)
		}

		//print slice with contents of file
		//for _, str := range s {
		//	println(str)
		//}
	}
	return s
}

func IsHumanReadable(file string) bool {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	b := make([]byte, 4)
	_, err = f.Read(b)
	if err != nil {
		panic(err)
	}
	// text := string(b)
	// 60 = E, 76 = L, 70 = F
	// ELF check
	if (b[1] == 69) && (b[2] == 76) && (b[3] == 70) {
		return false
	}
	// check for crazy ass bytes in file
	// 0x00 = null byte
	// 0x01 = start of heading
	// 0x02 = start of text

	r := bufio.NewReader(f)
	for {
		if c, _, err := r.ReadRune(); err != nil {
			if err == io.EOF {
				break
			}
		} else {
			if string(c) == "\x00" || string(c) == "\x01" || string(c) == "\x02" {
				return false
			}
		}
	}
	return true
}

func GetDiff(file1, file2 string) (string, error) {
	// Read the contents of both files into memory
	content1, err := ioutil.ReadFile(file1)
	if err != nil {
		return "", err
	}
	content2, err := ioutil.ReadFile(file2)
	if err != nil {
		return "", err
	}

	// Split the file contents into lines
	lines1 := splitLines(string(content1))
	lines2 := splitLines(string(content2))

	// Perform the diff
	var output string
	var start1, start2, length int
	for i, j := 0, 0; i < len(lines1) || j < len(lines2); {
		if i < len(lines1) && j < len(lines2) && lines1[i] == lines2[j] {
			// Lines are the same
			i++
			j++
		} else {
			// Lines are different
			start1 = i
			start2 = j
			for i < len(lines1) && j < len(lines2) && lines1[i] != lines2[j] {
				i++
				j++
			}
			length = i - start1
			if i < len(lines1) || j < len(lines2) {
				// There is another hunk after this one
				length = min(length, min(len(lines1)-start1, len(lines2)-start2))
			}
			output += getHunk(lines1, lines2, start1, start2, length)
		}
	}

	return output, nil
}

func splitLines(text string) []string {
	var lines []string
	start := 0
	for i, c := range text {
		if c == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

func getHunk(lines1, lines2 []string, start1, start2, length int) string {
	var output string
	output += fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", start1+1, length, start2+1, length)
	for i := start1; i < start1+length; i++ {
		output += fmt.Sprintf("-%s\n", lines1[i])
	}
	for i := start2; i < start2+length; i++ {
		output += fmt.Sprintf("+%s\n", lines2[i])
	}
	return output
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func DirSanityCheck() {
	//check if the homedir directory exists
	if _, err := os.Stat(dirhome); os.IsNotExist(err) {
		os.Mkdir(dirhome, 0700)
	}
	//check if the dirforbackups directory exists
	if _, err := os.Stat(dirforbackups); os.IsNotExist(err) {
		os.Mkdir(dirforbackups, 0700)
	}
	//check if the indexfile exists
	if _, err := os.Stat(indexfile); os.IsNotExist(err) {
		os.Create(indexfile)
	}
}

func VerifyRunIntegrity() {
	//EstablishPersistance() and VerifyRunIntegrity() must have a symbiotic relationship
	//because they are two halves of the same coin.  VerifyRunIntegrity() will check to
	//see if the persistence mechanism is still in place, and if not, it will re-establish
	//it.  This is to ensure that the persistence mechanisms are always in place.
	DirSanityCheck()

	Dirs := []string{dirhome, dirforlogging, dirforbackups}

	for _, dir := range Dirs {
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				os.Mkdir(dir, 0600)
			} else {
				panic(err)
			}
		}

		stats, err := os.Stat(dir)
		if err != nil {
			panic(err)
		}

		if stats.Mode().Perm() != 0600 {
			os.Chmod(dir, 0600)
		}
		fdir, _ := os.ReadDir(dir)
		for _, f := range fdir {
			fpath := filepath.Join(dir, f.Name())
			stats, err := os.Stat(fpath)
			if err != nil {
				panic(err)
			}
			if stats.Mode().Perm() != 0600 {
				os.Chmod(fpath, 0600)
			}
		}
	}
}
