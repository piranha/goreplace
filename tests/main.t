Go Replace tests:

  $ START_DIR=$PWD && cd $TESTDIR && cd ./.. && go build -o $START_DIR/gr && cd $START_DIR
  $ alias gr="$START_DIR/gr -c"

Usage:

  $ gr
  Usage:
    gr [OPTIONS] string-to-search
  
  Ignoring files bigger than 10MB
  General ignorer
  
  Application Options:
    -r, --replace=RE        replace found substrings with RE
        --force             force replacement in binary files
        --dry-run           prints replacements without modifying files
    -i, --ignore-case       ignore pattern case
    -s, --singleline        ^/$ will match beginning/end of line
    -p, --plain             treat pattern as plain text
    -x, --exclude=RE        exclude filenames that match regexp RE (multi)
    -o, --only=RE           search only filenames that match regexp RE (multi)
    -I, --no-autoignore     do not read .git/.hgignore files
    -b, --big-file=SIZE     ignore files bigger than SIZE (use suffixes: k, M)
    -B, --no-bigignore      do not ignore big files at all
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

Check that a dry run can be executed:

  $ mkdir dry-run && cd dry-run
  $ echo 'adc' > a.txt
  $ echo 'def' > b.txt
  $ gr 'a.c' --replace 'cba' --dry-run
  Searching for: a.c
  Replacing with: cba
  a.txt
    - adc
    + cba
    1 change
  $ cat a.txt
  adc
  $ cat b.txt
  def
  $ cd ..

Check that a find/replace executes correctly:

  $ mkdir find-replace && cd find-replace
  $ echo 'abc\nadc\ndef\nxyz' > abc.txt
  $ echo 'def\nadc\nxyz' > def.txt
  $ gr 'a(.)c' --replace 'c${1}a'
  abc.txt
    - abc
    + cba
    - adc
    + cda
    2 changes
  def.txt
    - adc
    + cda
    1 change
  $ cat abc.txt
  cba
  cda
  def
  xyz
  $ cat def.txt
  def
  cda
  xyz
  $ cd ..

