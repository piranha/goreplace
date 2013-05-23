Go Replace tests:

  $ go build goreplace
  $ gr=$(pwd)/goreplace
  $ alias noc="perl -pe 's/\e\[?.*?[\@-~]//g'"

Usage:

  $ $gr
  Usage of goreplace *: (glob)
  \tgr [OPTS] string-to-search (esc)
  
  General ignorer
  Options:
    -i     --ignore-case    ignore pattern case
    -n     --filename       print only filenames
    -x RE  --exclude=RE     exclude files that match the regexp from search
    -o RE  --only=RE        include only files that match this regexp
    -s     --singleline     match on a single line (^/$ will be beginning/end of line)
    -p     --plain          search plain text
    -r     --replace=       replace found substrings with this string
           --force          force replacement in binary files
    -V     --version        show version and exit
    -I     --no-autoignore  do not read .git/.hgignore files
    -v     --verbose        be verbose (show non-fatal errors, like unreadable files)
           --help           show usage message
  

Find a string in a file:

  $ mkdir one && cd one
  $ echo "test" > qwe
  $ $gr st | noc
  qwe
  1:test
  $ cd ..

Check that fnmatch-style gitignore patterns are handled:

  $ mkdir fnmatch && cd fnmatch
  $ mkdir .git
  $ mkdir -p one/two
  $ echo test > one/two/three
  $ $gr st | noc
  one/two/three
  1:test
  $ echo "one/*" > .gitignore
  $ $gr st | noc
  $ cd ..  
