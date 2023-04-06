package main

import (
	"flag"
	"os"

	"github.com/xFaraday/filehaunt/filelib"
	"github.com/xFaraday/filehaunt/log"
)

func usage() {
	println(`
	FileHaunt Usage:
		-file    <File|Directory>   : Specify file to backup or verify
		-overwrite                  : Specify overwrite flag to overwrite existing backup  (default: true)
		-backup                     : Switch for file backup                               (default: false)
		-verify                     : Switch for file verification                         (default: false)
	
	Example:
		BACKUP /etc/passwd:
		./filehaunt -file /etc/passwd -backup
		
		VERIFY /etc/passwd:
		./filehaunt -file /etc/passwd -verify

		VERIFY ALL FILES:
		./filehaunt -verify
	`)
}

func main() {
	if os.Getegid() != 0 {
		println("You must be root to run this program.")
		os.Exit(1)
	}

	filelib.VerifyRunIntegrity()
	log.InitLogger()

	var (
		backup    bool
		verify    bool
		file      string
		overwrite bool
	)

	flag.StringVar(&file, "file", "", "File path for backup or verify")
	flag.BoolVar(&backup, "backup", false, "Switch for file backup")
	flag.BoolVar(&overwrite, "overwrite", true, "Switch for file overwrite")
	flag.BoolVar(&verify, "verify", false, "Switch for file verification")
	flag.Parse()

	if !verify && !backup {
		usage()
		println("You must specify either -backup or -verify")
		os.Exit(1)
	}
	if backup && file == "" {
		usage()
		println("You must specify a file to backup")
		os.Exit(1)
	} else if backup && file != "" {
		filelib.RestoreController(file, overwrite)
	}
	if verify && file == "" {
		filelib.VerifyFiles()
	} //else if verify && file != "" {
	//filelib.VerifyFile(file)
	//}
}
