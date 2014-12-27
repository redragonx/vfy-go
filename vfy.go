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
* DIR - Directory in original missing from backup.
* FILE - File in original missing from, or different, in backup.
* SKIP - Skipping directory specified by --ignore.
* SYMMIS - Symlink mismatch (one is a symlink, one is a regular file, etc.).
* SYMLINK - Symlink to directory skipped and not not following (no --follow).
* DIFFS - Not recursing into dir because it is on a different filesystem.
* ERROR - Error reading file or directory.
* DEBUG - Debug information only shown when called with --verbose.
*/

import (
	"fmt"
	"log"
	"os"
	"io"
	"io/ioutil"
	"path/filepath"
	"bytes"
	"crypto/rand"
	"math/big"
	"math"
	"syscall"
	flag "github.com/ogier/pflag"
)

// the final report type
type resultSummary struct {
	diffCount    int
	itemCount    int
	skippedCount int
	errorCount   int
	symLinkError int
	symMisMatch  int
}

type dirNotFoundError struct {
	Dir         string
}


var sampleSize = 32

// ------------------------------------------------------------------------ //
var origDir = ""
var backupDir = ""

var verbose = flag.BoolP("verbose", "v", false, "Print what is being done")
var machine = flag.BoolP("machine", "m", false, "Output summary in machine-readable format")
var followSymlinks = flag.BoolP("follow", "f", false, "Follow symlinks")
var oneFilesystem = flag.BoolP("one-filesystem", "x", false, "Stay on one filesystem (in <original>)")
var ignoreDir = flag.StringP("ignore", "i", "", "Don't process DIR")
var sampleCount = flag.Int64P("samples", "s", 0, "Comparison sample count [default: 0]")
var help = flag.BoolP("help", "h", false, "Display this screen")

// ------------------------------------------------------------------------ //

// setup a resultsummary object to print later
var globalResultSummary = new(resultSummary)

func visit(path string, f os.FileInfo, err error) error {

	globalResultSummary.itemCount += 1

	original := path
	backup := filepath.Join(backupDir, filepath.Base(path))

	if *verbose {
		fmt.Printf("DEBUG: Comparing [%s] to [%s] \n", original, backup)
	}

//	if *ignoreDir != original || 
	// Make sure both directories exist
	folder1Stat, folder1Err := os.Stat(original)
	folder2Stat, folder2Err := os.Stat(backup)

	if folder1Err != nil {
		return folder1Err;
	}

	if folder2Err != nil {
		return folder2Err;
	}

	// are both folders here?
	if !(isDirOrFile(folder1Stat) == "directory" && isDirOrFile(folder2Stat) == "directory") {
		fmt.Printf("DIR [%s] not found in [%s]", original, backup)

		globalResultSummary.diffCount += 1
		itemCount := countItems(original)
		globalResultSummary.itemCount += itemCount
		globalResultSummary.diffCount += itemCount
		return fmt.Errorf("DirError: This directory could not be read %s in %s", original, backup);
	}
	

	// if both folders are here.

	// This check is independent of whether or not the path is a directory or
	// a file. If either is a symlink, make sure they are both symlinks, and
	// that they link to the same thing.

	if isSymLink(original) || isSymLink(backup) {
		if isSymLink(original) && isSymLink(backup) {

			symlink1, symErr1 = filepath.EvalSymlinks(original);
			symlink2, symErr2 = filepath.EvalSymnlinks(backup);

			if symErr1 || symErr2 != nil {			
				return fmt.Errorf("SYMMIS: Syslink match [%s] and [%s]", original, backup)
			}

			if symlink1 != symlink2 {

			}
			
			fmt.Printf("SYMMIS: Syslink match [%s] and [%s]", original, backup)


		}
	}
	stat, err := os.Stat("/etc")
	if err != nil {
		log.Fatal(err)
	}
	dev := stat.Sys().(*syscall.Stat_t).Dev


	if(sameFile(original, backup)) {
		globalResultSummary.itemCount += 1;
	}

	return nil
}

func compareDirs(relative string) {
	err := filepath.Walk(relative, visit)


	fmt.Printf("filepath.Walk() returned %v\n", err)
}

func sameFile(fileA, fileB string) bool {

	// both files exists
	fileAOK, fileAErr := doesFileExist(fileA)
	fileBOK, fileBErr := doesFileExist(fileB)

	if fileAOK != true && fileAErr != nil && fileBOK != true && fileBErr != nil { 
		globalResultSummary.errorCount += 1
		return false
	}

	// both files are the same size
	fileASize, aSizeErr := getFileSize(fileA)
	fileBSize, bSizeErr := getFileSize(fileB)

	if aSizeErr != nil {
		fmt.Print("Can't get file size for" + aSizeErr.Error())
		globalResultSummary.errorCount += 1
		return false
	}
	
	if bSizeErr != nil {
		fmt.Print("Can't get file size for" + bSizeErr.Error())
		globalResultSummary.errorCount += 1
		return false
	}
	
	if fileASize != fileBSize {
		globalResultSummary.errorCount += 1
		return false
	}

	// read both files
	f1, f1err := os.Open(fileA)
	f2, f2err := os.Open(fileB)

	defer f1.Close()
	defer f2.Close()

	if f1err != nil {
		fmt.Print(f1err)
		globalResultSummary.errorCount += 1
		return false
	}

	if f2err != nil {
		fmt.Print(f2err)
		globalResultSummary.errorCount += 1
		return false
	}

	same := true
	
	// random sample test
	for i := int64(0); i < *sampleCount; i++ {
		startAtByte, randErr := getRandomNumberWithMax(fileASize)

		if randErr != nil {
			log.Fatal(randErr)
		}
		
		// get a random number of bytes to test...
		testBytesLength := math.Min(float64(fileASize), (float64(startAtByte) + float64(sampleSize))) - float64(startAtByte) + 1.0
		
		aSample := make([]byte, int(testBytesLength))
		bSample := make([]byte, int(testBytesLength))

		f1ReadByteNum, f1ReadErr := io.ReadAtLeast(f1, aSample, int(testBytesLength))
		f2ReadByteNum, f2ReadErr := io.ReadAtLeast(f2, bSample, int(testBytesLength))

		// file data is different
		if f1ReadErr != nil {
			fmt.Println(f1ReadErr)
			globalResultSummary.errorCount += 1
			return false
		}
		
		if f2ReadErr != nil {
			fmt.Println(f2ReadErr)
			globalResultSummary.errorCount += 1
			return false
		}

		if (f1ReadByteNum != int(testBytesLength)) &&
	       (f2ReadByteNum != int(testBytesLength)) {

			globalResultSummary.errorCount += 2
			return false
		}

		// check the actual sample data
		if(bytes.Equal(aSample, bSample)) {
			break
	} else {
			return false
		}
	} // end of random test  loop
	return same
}

func getRandomNumberWithMax(max int64) (int64, error) {

	maxBigInt := big.NewInt(max)
	i, err := rand.Int(rand.Reader, maxBigInt)

	if err != nil {
		return 0, err
	}

	return i.Int64(), nil;
}

func getFileSize(file string) (int64, error) {
	
	fi, err := os.Stat(file)

	if err != nil {
		return 0, err
	}

	return fi.Size(), nil
}

func isSymLink(file string) (bool) {
	
	fi, err := os.Lstat(file)

	if err != nil {
		fmt.Println(err)
		return false
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

func countItems(dir string) int {
	if *verbose {
		fmt.Printf("DEBUG: Counting files in [%s]", dir)
	}

	count := 0

	dirItems, folderErr := ioutil.ReadDir(dir)

	if folderErr != nil {
		fmt.Printf("Unable to read %s, the error was %s", dir, folderErr.Error())
		globalResultSummary.errorCount += 1
	}

	for _, f := range dirItems {
		count += 1
		
		if f.IsDir() {
			fullPath := filepath.Join(dir, f.Name())
			count += countItems(fullPath)
		}
	}
	return count
}

// func trapSignals(chan signalChannel) {
// 	
// 	sig := <-signalChannel
//         switch sig {
//         case os.Interrupt:
//             //handle SIGINT
//        // case syscall.SIGTERM:
//             //handle SIGTERM
//         }	
// }


func main() {
usage := `Backup verify script.

Usage:
  vfy [options] <origDir> <backupDir>
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

	flag.Parse()
	
	if *help {
		fmt.Print(usage)
		os.Exit(1)
	}

	cmdLineStuff := flag.Args()

	if len(cmdLineStuff) < 2 {
		log.Fatal("You must specify original and backup folders.")
	}

	origDir = cmdLineStuff[0]
	backupDir = cmdLineStuff[1]

	// Does these locations exist?
	origDirStats, statErr := os.Stat(origDir)
	backupDirStats, statErr2 := os.Stat(backupDir)
	
	if(statErr != nil) {
		log.Fatalf("%s cannot be read", origDir)
	}
	
	if(statErr2 != nil) {
		log.Fatalf("%s cannot be read", backupDir)
	}

	if isDirOrFile(origDirStats) != "directory" {
		log.Fatalf("%s is not a directory.", origDir)
	}
	
	if isDirOrFile(backupDirStats) != "directory" {
		log.Fatalf("%s is not a directory.", origDir)
	}

	compareDirs(origDir)

}
