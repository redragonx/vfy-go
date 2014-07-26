package main

/*
* Author: Stephen Chavez
* WWW: dicesoft.net
* Date: June 30, 2014
* License: Public domain / Do whatever you want.
*
* Backup validator script. Compares two folders "original" and "backup".
* Alerts the user of any files or directories that are in "original" but not in
* "backup" (extra files in "backup" are ignored). If a file exists in both
* "original" and "backup," they are compared by checking their lengths and by a
* random sample of their contents, and the user is alerted if they differ.
*
* Output prefixes:
* DIR: - Directory in original missing from backup.
* FILE: - File in original missing from, or different, in backup.
* SKIP: - Skipping directory specified by --ignore.
* SYMLINK: - Symlink to directory skipped and not not following (no --follow).
* DIFFS - Not recursing into dir because it is on a different filesystem.
* ERROR: - Error reading file or directory.
* DEBUG: - Debug information only shown when called with --verbose.
 */

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"math/rand"
	"github.com/docopt/docopt-go"
)

// the final report type
type resultSummary struct {
	diffCount    int
	itemCount    int
	skippedCount int
	errorCount   int
	symLinkError int
}

func (rs *resultSummary) addDiffCount(num int) {
	rs.diffCount += num
}

func (rs *resultSummary) addItemCount(num int) {
	rs.itemCount += num
}

func (rs *resultSummary) addSkippedCount(num int) {
	rs.skippedCount += num
}

func (rs *resultSummary) addErrorCount(num int) {
	rs.errorCount += num
}

var argOptions map[string]interface{};
var globalResultSummary resultSummary;

func compareRootFolders() {
	// get the stats for these files

	if origDirStats, err := os.Stat(argOptions["<orig_dir>"].(string)); err == nil {
		if backupDirStats, err := os.Stat(argOptions["<backup_dir>"].(string)); err == nil {
		
			// check if the user gavee actual files, if so, quit program with an error

			if dirOK := isDirOrFile(origDirStats); dirOK == "file" {
				log.Fatal("You gave a file as <orig_dir>, try again.")
			}
			
			if dirOK := isDirOrFile(backupDirStats); dirOK == "file" {
				log.Fatal("You gave a file as <backup_dir>, try again!")
			}


		} else {
			log.Fatal("<backup_dir> " + err.Error())
		}
	} else {
		log.Fatal("<backup_dir> " + err.Error())
	}

}

func sameFile(fileA, fileB string) bool {

	// both files exists
	fileAOK, fileAErr := doesFileExist(fileA)
	fileBOK, fileBErr := doesFileExist(fileB)

	if fileAOK != true && fileAErr != nil && fileBOK != true && fileBErr != nil { 
		return false
	}

	// both files are the same size
	fileASize, aSizeErr := getFileSize(fileA)
	fileBSize, bSizeErr := getFileSize(fileB)

	if aSizeErr != nil && bSizeErr != nil {
		return false
	} else if fileASize != fileBSize {
		return false
	}

	sampleNumber, convertErr := strconv.Atoi(argOptions["COUNT"].(string))

	// check if the user entered a bad sample number
	if convertErr != nil {
		log.Fatal("The -s argument was bad.")
	}

	same := true;

	// random sample test
	for i := 0; i < sampleNumber; i++ {


	}


}


func getFileSize(file string) (int64, error) {
	
	fi, err := os.Stat(file) 

	if err != nil {
		return 0, err
	}

	return fi.Size(), nil
}
func isSymLink(file string) bool {
	
	fi, err := os.Lstat(file) 

	if err != nil {
		fmt.Println(err)
		globalResultSummary.addErrorCount(1)
	}

	if fi.Mode() & os.ModeSymlink == os.ModeSymlink {
		return true
	}
	
	return false

}
 
// This function returns true and nil as the returned values, only if the file
// exists. And the function did not return any type of error.
// it returns any other error if the error returned by os.Stat is not the
// expected "does not exist" error.
func doesFileExist(file string) (bool, error) {
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func checkContents(relative string) {


}

func isDirOrFile(fi os.FileInfo) string {

	switch mode := fi.Mode(); {
	case mode.IsDir():
		// do directory stuff
		return "directory"
	case mode.IsRegular():
		// do file stuff
		return "file"
	}

	return ""
}

func main() {
	usage := `Backup verify script.

Usage:
  vfy [options] <orig_dir> <backup_dir>
  vfy (-h | --help)

This program compares two directories recursively, and alerts the user of any
differences. It compares files by size and **optionally** by a random sample of
contents. The results are summarized into a difference percentage so it can be
used to easily determine if a backup is valid and recent. 

Options:
  -v, --verbose                    Print what is being done
  -m, --machine                    Output summary in machine-readable format
  -f, --[no-]follow                Follow symlinks
  -x, --one-filesystem             Stay on one file system (in <original>)
  -c, --count                      Count files in unmatched directories
  -i, --ignore DIR                 Don't process DIR
  -s, --samples COUNT              Comparison sample count [default: 0]
  -h, --help                       Display this screen`

    var argErr error
	argOptions, argErr =  docopt.Parse(usage, nil, true, "Go verify 1.0", false)
	fmt.Println(argOptions)
	fmt.Println(argOptions["<orig_dir>"])

	if argErr != nil {
		fmt.Println("argError: " + argErr.Error())
	}

	if argOptions["<orig_dir>"] == nil || argOptions["<backup_dir>"] == nil {
		log.Fatal("Please specify backup and original folders")
	}

	// end of docopt stuff.

	// setup a resultsummary object to print later
	globalResultSummary := new(resultSummary)

	compareRootFolders()

	
}

