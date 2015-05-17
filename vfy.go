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
* Code based on https://github.com/defuse/backup-verify 
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
	"os/signal"
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
	diffPct 	 string
}


func (r *resultSummary) getMachineText() string {
	return fmt.Sprintf("SUMMARY: items: %d, diff:%d, diffpct:%s skip:%d, err:%d, symErr:%d, symMisMatchErr:%d ",
		   r.itemCount, r.diffCount, r.diffPct, r.skippedCount, r.errorCount, r.symLinkError, r.symMisMatch )

}

func (r *resultSummary) getHumanReadableText() string {
	similarities := r.itemCount - r.diffCount

	return fmt.Sprintf("\nSUMMARY: \n\t Items processed: %d \n\t Differences: %d (%s%%) \n\t Similarities: %d \n\t Skipped: %d \n\t Errors: %d \n\t Symlink Errors: %d \n\t SymMisMatch Errors: %d \n",
		   r.itemCount, r.diffCount, r.diffPct, similarities, r.skippedCount, r.errorCount, r.symLinkError, r.symMisMatch )
}


// The number of bytes to compare during each random sample comparison.
var sampleSize = 32

var userSource1 = ""
var userSource2 = ""


// setup a resultsummary object to print later
var globalResultSummary = new(resultSummary)

// ------------------------------------------------------------------------ //

var verbose = flag.BoolP("verbose", "v", false, "Print what is being done")
var machine = flag.BoolP("machine", "m", false, "Output summary in machine-readable format")
var oneFilesystem = flag.BoolP("one-filesystem", "x", false, "Stay on one filesystem (in <original>)")
var sampleCount = flag.Int64P("samples", "s", 0, "Comparison sample count [default: 0]")
var help = flag.BoolP("help", "h", false, "Display this screen")

// ------------------------------------------------------------------------ //


func visit(relative string)  {


	original := filepath.Join(userSource1, relative)
	backup := filepath.Join(userSource2, relative)

	if *verbose {
		fmt.Printf("DEBUG: Comparing [%s] to [%s] \n", original, backup)
	}

//	if *ignoreDir != original || 
	// Make sure both directories exist
	file1Stat, file1Err := os.Stat(original)
	file2Stat, file2Err := os.Stat(backup)

	if file1Err != nil {
		globalResultSummary.diffCount += 1
		globalResultSummary.errorCount += 1
		fmt.Printf("[%s] not a valid folder/file \n", original)

		return
	}
	
	if file2Err != nil {
		globalResultSummary.diffCount += 1
		globalResultSummary.errorCount += 1
		fmt.Printf("[%s] not a valid folder/file \n", backup)

		return
	}
	
	// Stay on one filesystem if told to do so...

	

	// are both folders here?
	if isDirOrFile(file1Stat) == "directory" {
		if *verbose {
			fmt.Print("\n In folder checking code\n\n")
		}

		if isDirOrFile(file2Stat) != "directory"  {
		fmt.Print(file2Err.Error())

		if *verbose {
			fmt.Print("\n In folder checking code: SECOND IF STATEMENT")
			fmt.Printf("DIR [%s] not found in [%s]", original, backup)
		}

			globalResultSummary.diffCount += 1
			
			itemCount := countItems(original)
			globalResultSummary.itemCount += itemCount
			globalResultSummary.diffCount += itemCount

			return
		}

		
		// Read child folder's contents
		files, folderReadErr := ioutil.ReadDir(original)
		
		if folderReadErr != nil {
			globalResultSummary.diffCount += 1
			globalResultSummary.errorCount += 1
			fmt.Printf("Cannot read folder contents: [%s]  \n", original)

			return
		}

		for _, item := range files {

			if item.Name() == "." || item.Name() == ".." {
				continue;
			}
			
			globalResultSummary.itemCount += 1

			origChildItemPath := filepath.Join(original, item.Name())
			backupChildItemPath := filepath.Join(backup, item.Name())
			
			
			// This check is independent of whether or not the path is
			// a directory or a file. If either is a symlink, make sure they are
			// both symlinks, and that they link to the same thing.

			if isSymLink(origChildItemPath) || isSymLink(backupChildItemPath) {
				if isSymLink(origChildItemPath) &&
					isSymLink(backupChildItemPath) {

					symlink1, symErr1 := filepath.EvalSymlinks(origChildItemPath);
					symlink2, symErr2 := filepath.EvalSymlinks(backupChildItemPath);

					if symErr1 != nil {			
						fmt.Printf("SYMMIS: Syslink read error [%s] \n",
						origChildItemPath)
						
						globalResultSummary.symLinkError += 1
						return
					}
					
					if symErr2 != nil {			
						fmt.Printf("SYMMIS: Syslink read error [%s] \n",
						backupChildItemPath)

						globalResultSummary.symLinkError += 1
						return
					}

					// SYMLINK MISMATCH
					if symlink1 != symlink2 {
						fmt.Printf("SYMMIS: Syslink mismatch [%s] and [%s] \n",
							origChildItemPath,
							backupChildItemPath)


						// Count the missing file or directory.
						globalResultSummary.diffCount += 1
						globalResultSummary.symMisMatch += 1

						// If the orignal symlink was a directory, then the 
						// backup is missing that directory, PLUS all of that 
						// directory's contents.

						if file1Stat.IsDir() {
							itemCount := countItems(origChildItemPath)
							globalResultSummary.itemCount += itemCount
							globalResultSummary.diffCount += itemCount
						}
						return
					}
				}
			}

			
			if item.IsDir() {
					
			checkFileSystem(userSource1, origChildItemPath);
			
			visit(filepath.Join( relative, item.Name()))
			} else {
				if !sameFile(origChildItemPath, backupChildItemPath) {
					globalResultSummary.diffCount += 1;
					fmt.Printf( "FILE: [%s] not found at," +
						"or doesn't match [%s] \n", origChildItemPath,
						backupChildItemPath)
				}
			}
		} // end for loop
	}

}

func compare(relative string) {
	visit(relative)
}

func checkFileSystem(fileA, fileB string) {

	outerDevStat, outerDevStatErr := os.Stat(fileA)
	innerDevStat, innerDevStatErr := os.Stat(fileB)

	if outerDevStatErr != nil {
		globalResultSummary.skippedCount += 1
		fmt.Print(outerDevStatErr)
		return
	}
	
	if innerDevStatErr != nil {
		globalResultSummary.skippedCount += 1
		fmt.Print(innerDevStatErr)
		return
	}

	outerDev := outerDevStat.Sys().(*syscall.Stat_t).Dev
	innerDev := innerDevStat.Sys().(*syscall.Stat_t).Dev

	if outerDev != innerDev && *oneFilesystem {
		globalResultSummary.skippedCount += 1
		fmt.Printf("DIFFERS: [%s] is on a different file system than [%s]." +
			"Skipped \n",
			fileA,
			fileB)
		return
	}
}

func sameFile(fileA, fileB string) bool {

	// both files exists
	fileAOK, fileAErr := doesFileExist(fileA)
	fileBOK, fileBErr := doesFileExist(fileB)

	if fileAOK != true &&
		fileAErr != nil &&
		fileBOK != true &&
		fileBErr != nil {
	
		globalResultSummary.errorCount += 1
		return false
	}

	// both files are the same size
	fileASize, aSizeErr := getFileSize(fileA)
	fileBSize, bSizeErr := getFileSize(fileB)

	if aSizeErr != nil {
		fmt.Print("Can't get file size for" + aSizeErr.Error() + "\n")
		globalResultSummary.errorCount += 1
		return false
	}
	
	if bSizeErr != nil {
		fmt.Print("Can't get file size for" + bSizeErr.Error() + "\n")
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
		testBytesLength := math.Min(float64(fileASize),
			(float64(startAtByte) + float64(sampleSize))) -
									float64(startAtByte) + 1.0
		
		aSample := make([]byte, int(testBytesLength))
		bSample := make([]byte, int(testBytesLength))

		f1ReadByteNum, f1ReadErr := io.ReadAtLeast(f1,
			aSample,
			int(testBytesLength))

		f2ReadByteNum, f2ReadErr := io.ReadAtLeast(f2,
			bSample,
			int(testBytesLength))

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
	} // end of random test loop
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

	if *verbose {
		fmt.Print( "\nin isSymlink" + file + "\n")
	}

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
		fmt.Printf("Unable to read %s, the error was %s",
		dir,
		folderErr.Error() +"\n")
		
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

func printSummary() {
	globalResultSummary.diffPct = fmt.Sprintf("%.f",
		(float64(globalResultSummary.diffCount) /
		float64(globalResultSummary.itemCount) * 100));

	if *machine {
		fmt.Print(globalResultSummary.getMachineText());
	} else {
		fmt.Print(globalResultSummary.getHumanReadableText())
	}
}

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
  -x, --one-filesystem             Stay on one file system (in <original>)
  -s, --samples COUNT              Comparison sample count [default: 0]
  -h, --help                       Display this screen`


	flag.Parse()
	
	if *help {
		fmt.Print(usage)
		fmt.Println();
		os.Exit(1)
	}

	cmdLineStuff := flag.Args()
	
	if len(cmdLineStuff) < 2 {
		log.Fatal("You must specify original and backup folders.")
	}
	
	sig := make(chan os.Signal, 1)
    signal.Notify(sig, os.Interrupt, os.Kill)

	userSource1 = cmdLineStuff[0]
	userSource2 = cmdLineStuff[1]

	// Does these locations exist?
	origDirStats, statErr := os.Stat(userSource1)
	backupDirStats, statErr2 := os.Stat(userSource2)
	
	if(statErr != nil) {
		log.Fatalf("%s cannot be read", userSource1)
	}
	
	if(statErr2 != nil) {
		log.Fatalf("%s cannot be read", userSource2)
	}

	if isDirOrFile(origDirStats) != "directory" {
		log.Fatalf("%s is not a directory.", userSource1)
	}
	
	if isDirOrFile(backupDirStats) != "directory" {
		log.Fatalf("%s is not a directory.", userSource2)
	}


	 compare("")
	 printSummary();

}
