Go Replace tests:

  $ go build -o gr goreplace 2> /dev/null || go build -o gr github.com/piranha/goreplace
  $ CURRENT=$(pwd)
  $ alias gr="$CURRENT/gr -c"

Usage:

  $ gr
  Usage:
    gr [OPTIONS] string-to-search
  
  General ignorer
  
  Application Options:
    -r, --replace=RE        replace found substrings with RE
        --force             force replacement in binary files
    -i, --ignore-case       ignore pattern case
    -s, --singleline        ^/$ will match beginning/end of line
    -p, --plain             treat pattern as plain text
    -x, --exclude=RE        exclude filenames that match regexp RE (multi)
    -o, --only=RE           search only filenames that match regexp RE (multi)
    -I, --no-autoignore     do not read .git/.hgignore files
    -B, --no-bigignore      do not ignore files bigger than 10M
    -f, --find-files        search in file names
    -n, --filename          print only filenames
    -v, --verbose           show non-fatal errors (like unreadable files)
    -c, --no-colors         do not show colors in output
    -N, --no-group          print file name before each line
    -V, --version           show version and exit
    -h, --help              show this help message

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

Check that .* matches only files starting with dot:

  $ mkdir dotstar && cd dotstar
  $ mkdir .git
  $ echo '.*' > .gitignore
  $ echo qwe > .ignore
  $ echo qwe > notignore
  $ gr qwe
  notignore
  1:qwe
  $ cd ..

Check that percents don't do anything bad:

  $ mkdir percent && cd percent
  $ echo 'hello %username%' > one
  $ gr user
  one
  1:hello %username%
  $ cd ..
