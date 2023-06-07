package filelib

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blockloop/scan"
	"github.com/fsnotify/fsnotify"
	"github.com/xFaraday/filehaunt/db"
	"go.uber.org/zap"
)

var (
	indexfile = "/opt/filehaunt/index.safe"
)

type fileindex struct {
	filepath   string
	name       string
	backupfile string
	backuptime string
	hash       string
}

/*
	Add edge case check in VerifyFiles() to see if the file has been deleted
		- if so, unzip compressed file back to original spot
		- if not, proceed as normal
*/

func WatchDir(dir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		zap.S().Error("Unable to watch directory: ", dir)
	}

	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Create == fsnotify.Create {
					zap.S().Info("File Created: ", event.Name, " in Dir: ", dir)
					os.Remove(event.Name)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				zap.S().Error(err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}
	<-done

}

func VerifyFiles() {
	countbackedfiles := db.CountRows("fileindex")
	if countbackedfiles != 0 {
		conn := db.DbConnect()
		rows, err := conn.Query("SELECT * FROM fileindex")
		if err != nil {
			zap.S().Error("Unable to query fileindex table in Database")
		}
		defer rows.Close()

		var fileinfo []fileindex
		err1 := scan.Rows(&fileinfo, rows)
		if err1 != nil {
			zap.S().Error(err1)
		}

		for _, f := range fileinfo {
			if _, err := os.Stat(f.filepath); err != nil {
				if os.IsNotExist(err) {
					CompressedBackup := dirforbackups + f.backupfile
					TmpCmpFile, _ := os.Create("/tmp" + f.name + ".tmp")
					RevertCompressedFile, _ := os.Open(CompressedBackup)
					Decompress(RevertCompressedFile, TmpCmpFile)
					oGfile, _ := os.Create(f.filepath)

					zap.S().Warn("File:" + f.filepath + " has been deleted, restoring from backup")

					OverWriteModifiedFile(oGfile.Name(), TmpCmpFile.Name())
					os.Remove(TmpCmpFile.Name())
				} else {
					panic(err)
				}
			}

			fCurrentStats, _ := CheckFile(f.filepath)
			if fCurrentStats.Hash != f.hash {
				CompressedBackup := dirforbackups + f.backupfile
				TmpCmpFile, _ := os.Create("/tmp" + f.name + ".tmp")
				RevertCompressedFile, _ := os.Open(CompressedBackup)
				Decompress(RevertCompressedFile, TmpCmpFile)

				//FIGURE OUT IF TXT FILE THEN TRY TO GET DIFF
				diff, _ := GetDiff(f.filepath, TmpCmpFile.Name())
				if diff == "binary, no diff" {
					zap.S().Warn("File:" + f.filepath + " has been modified, but is binary, no diff available")
				} else {
					zlog := zap.S().With(
						"file", f.filepath,
						"diff", diff,
					)
					zlog.Warn("File has been modified, diff below")
				}

				h := sha256.New()
				h.Write([]byte(diff))
				Enc := base64.StdEncoding.EncodeToString(h.Sum(nil))

				stmt := db.InsertIntoTable("filechanges")
				_, err := stmt.Exec(f.filepath, fCurrentStats.Time, diff, Enc)
				if err != nil {
					zap.S().Error("Unable to log diff in table: filechanges")
				}

				//actions once the difference is logged
				OverWriteModifiedFile(f.filepath, TmpCmpFile.Name())
				os.Remove(TmpCmpFile.Name())
				zap.S().Info("File: " + f.filepath + " has been restored to original state")
			}
		}
	}
}

/*
	Improve Compress and Decompress later:
		-> Add dictionary method for better compression
		-> Better manage encoders and decoders
*/

func BackFile(storename string, file string /*, mode int*/) {
	println("backing up file: " + file)
	OriginFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	CompressedFile, err := os.Create(dirforbackups + storename)
	if err != nil {
		panic(err)
	}

	PointData := bufio.NewReader(OriginFile)
	Compress(PointData, CompressedFile)

	defer OriginFile.Close()
	defer CompressedFile.Close()
}

func ExistsInIndex(indexfile string, file string) string {
	//strlist := OpenFile(indexfile)

	var count int

	conn := db.DbConnect()
	query := "SELECT COUNT(filepath) FROM fileindex where filepath = " + file
	if err := conn.QueryRow(query).Scan(&count); err != nil {
		zap.S().Warn("Unable to get count of table: fileindex")
	}

	if count != 0 {
		return "new"
	} else {
		return "newback"
	}
	/*
		for _, indexstr := range strlist {
			splittysplit := strings.Split(indexstr, "|-:-|")
			if splittysplit[0] == file {
				println("exact file exists in index")
				return "newback"
			}
		}
	*/
	//return "new"
}

func OverWriteModifiedFile(OriginalPath string, FileBackup string) {
	//delete original
	//call modified BackFile function
	os.Remove(OriginalPath)
	BytesToCopy, _ := os.Open(FileBackup)
	NewFile, _ := os.Create(OriginalPath)
	if _, err := io.Copy(NewFile, BytesToCopy); err != nil {
		panic(err)
	}
	defer BytesToCopy.Close()
	defer NewFile.Close()
}

func OverWriteBackup(storename string, file string) {
	f := OpenFile(indexfile)
	for _, indexstr := range f {
		splittysplit := strings.Split(indexstr, "|-:-|")

		if file == splittysplit[0] {
			os.Remove(dirforbackups + splittysplit[2])
			BackFile(splittysplit[2], file)
		}
	}
}

func BackDir(file string, overwrite bool) {
	fdir, _ := os.ReadDir(file)

	for _, f := range fdir {
		fpath := filepath.Join(file, f.Name())
		CreateRestorePoint(fpath, overwrite)
	}
}

func GenRandomName() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, 15)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func CreateRestorePoint(file string, overwrite bool) {
	stats, _ := CheckFile(file)
	VerifyRunIntegrity()
	if stats.Size != 0 {
		/*
			Index file format:
			Simple ->
			fullpath-:-filename w/extension-:-CompressedBackupName-:-LastModTime-:-hash
			Ex:
			/opt/memento/index.safe-:-index.safe-:-ADZOPRJ13SMF.zst-:-2021-01-01 00:00:00-:-9pN02HFtrhT4EGw+SdIECoj0HV8PBLY8qkZjwaKGRvo=
		*/
		//indexstr := strings.Split(file, "/")
		if stats.Hash == "directory" {
			BackDir(file, overwrite)
		} else {
			strsplit := strings.Split(file, "/")
			storename := strsplit[len(strsplit)-1]

			// /etc/passwd-:-passwd.txt-:-some date-:-hash
			backname := GenRandomName() + ".zst"
			indexstr := file + "|-:-|" + storename + "|-:-|" + backname + "|-:-|" + stats.Time + "|-:-|" + string(stats.Hash) + "\n"

			checkresult := ExistsInIndex(indexfile, file)

			switch checkresult {
			case "newback":
				if overwrite {
					zap.S().Info("Overwriting backup for file: " + file)
					//println("Overwriting previous backup of :" + file)
					OverWriteBackup(storename, file)
				} else {
					zap.S().Error("Skipping backup for file, overwrite set to n: " + file)
					println("overwrite is set to n, exiting")
					os.Exit(0)
				}
			case "new":
				appendfile, err := os.OpenFile(indexfile, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					panic(err)
				}
				appendfile.WriteString(indexstr)

				stmt := db.InsertIntoTable("fileindex")
				_, err1 := stmt.Exec(file, storename, backname, stats.Time, stats.Hash)
				if err1 != nil {
					zap.S().Fatal("Unable to insert metadata on file: ", file, " into database!")
				}

				defer appendfile.Close()

				zap.S().Info("BACKUP File: " + file + " has been backed up sucessfully")
				//println("BACKING UP FILE: " + file)

				BackFile(backname, file)
				//PostToServ(m)
			}
			//}
		}
	} else {
		println("Nothing to backup :(, file is empty")
	}
}

func RestoreController(file string, overwrite bool) {
	VerifyRunIntegrity()
	CreateRestorePoint(file, overwrite)
}
