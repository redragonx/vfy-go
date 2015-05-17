Backup verifier program in Go.  
=================

Abstract: 
---------

For my own benefit, I ported @defuse's [backup verifier
script](https://github.com/defuse/backup-verify) to Go. The lang from Google

See my [blog post](https://dicesoft.net/blog/go-backup-program.html) for my thoughts on making this. 

Description of the program:
----------------------------
Backup verify script.

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
  -h, --help                       Display this screen


How to install on your system
-----------------------------

1. Setup Go, you can read how [here](https://golang.org/doc/install)
2. run `go get https://github.com/redragonx/vfy-go`
3. cd into the src dir and run `go install`
4. If done properly, you can run the program anywhere.
