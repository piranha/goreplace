# goreplace

goreplace is a simple utility which can be used as replacement for grep + sed
combination in one of most popular cases - find files, which contain something,
possibly replace this with something else.

## Why

Why do thing which is done by grep, find, and sed? Well, for one - I grew tired
of typing long commands with pipes and ugly syntax. You want to search? Use
grep. Replace? Use find and sed! Different syntax, context switching,
etc. Switching from searching to replacing with goreplace is 'up one item in
history and add a replacement string', much simpler!

Besides, it's also faster than grep! Hard to believe, and it's a bit of cheating -
but goreplace by default ignores everything you have in your `.hgignore` and
`.gitignore` files, skipping binary builds and compiled byte-codes (which you
usually don't want to touch anyway).

This is my reason to use it - less latency doing task I'm doing often.

## Installation

Binary builds (64 bit):

 - [Linux](http://solovyov.net/files/gr-0.4.1-linux)
 - [OS X](http://solovyov.net/files/gr-0.4.1-osx)
 - [Windows](http://solovyov.net/files/gr-0.4.1-win.exe)

You can download latest release of goreplace from GitHub's [downloads]().

But it's suited to be installed via `go` tool, so you can do usual thing:

    go get github.com/piranha/goreplace

And you should be done. You have to have `$GOPATH` set for this to work (`go`
will put sources and generated binary there). Add `-u` flag there to update your
goreplace.

I prefer name `gr` to `goreplace`, so I link `gr` somewhere in my path (usually
in `~/bin`) to `$GOPATH/bin/goreplace`.

## Usage

Usage is pretty simple, you can just run `gr` to see help on options. Basically
you just supply regexp (or a simple string - it's a regexp always as well) as an
argument and goreplace will search for it in all files starting from the current
directory, just like this:

    gr somestring

Some directories and files can be ignored by default (`gr` is looking for your
`.hgignore`/`.gitignore` in parent directories), just run `gr` without any
arguments to see help message - it contains information about them.

If you need to replace found strings with something, just pass `-r replacement`
option and they will be replaced in-place. No backups are made (not that you
need them, right? You're using version control, aren't you?).  Unfortunately
only plain strings are supported as replacement, no regexp submatch support yet
(planned, though).

[downloads]: https://github.com/piranha/goreplace/downloads
