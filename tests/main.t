Go Replace tests:

  $ go build goreplace
  $ CURRENT=$(pwd)
  $ gr () {
  > $CURRENT/goreplace $@ | perl -pe 's/\e\[?.*?[\@-~]//g'
  > }

Usage:

  $ gr
  Usage:
    goreplace [OPTIONS] string-to-search
  
  General ignorer
  
  Application Options:
    -i, --ignore-case       ignore pattern case
    -n, --filename          print only filenames
    -f, --find-files        search for files and not for text in them
    -x, --exclude=RE        exclude files that match the regexp from search
    -o, --only=RE           search only in files that match the regexp
    -s, --singleline        match on a single line (^/$ will be begginning/end of
                            line)
    -p, --plain             search plain text
    -r, --replace=          replace found substrings with this string
    -I, --no-autoignore     do not read .git/.hgignore files
        --force             force replacement in binary files
    -v, --verbose           be verbose (show non-fatal errors, like unreadable
                            files)
    -V, --version           show version and exit
        --help              show this help message
    -c, --no-colors         do not show colors in output

Find a string in a file:

  $ mkdir one && cd one
  $ echo "test" > qwe
  $ gr st
  qwe
  1:test
  $ cd ..

Check that fnmatch-style gitignore patterns are handled:

  $ mkdir fnmatch && cd fnmatch
  $ mkdir .git
  $ mkdir -p one/two
  $ echo test > one/two/three
  $ gr test
  one/two/three
  1:test
  $ echo "one/*" > .gitignore
  $ gr test
  $ cd ..  

Check plain text searching:

  $ mkdir plain && cd plain
  $ echo '\d' > test
  $ gr '\d'
  $ gr -p '\d'
  test
  1:\d
  $ cd ..

Check that anchored directories are properly ignored:

  $ mkdir nested && cd nested
  $ mkdir .git
  $ mkdir -p one/two
  $ echo test > one/two/three
  $ echo "/one/two" > .gitignore
  $ gr test
  $ cd ..

