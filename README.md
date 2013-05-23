# Go Replace

Go Replace (gr) is a simple utility which can be used as replacement for grep +
sed combination in one of most popular cases - find files, which contain
something, possibly replace this with something else. Main points:

 - Uses `.hgignore`/`.gitignore` to skip files
 - Skips binaries
 - Familiar PCRE-like regexp syntax
 - Can perform replacements
 - Fast

## Why

Why do thing which is done by grep, find, and sed? Well, for one - I grew tired
of typing long commands with pipes and ugly syntax. You want to search? Use
grep. Replace? Use find and sed! Different syntax, context switching,
etc. Switching from searching to replacing with gr is 'up one item in
history and add a replacement string', much simpler!

Besides, it's also faster than grep! Hard to believe, and it's a bit of cheating -
but gr by default ignores everything you have in your `.hgignore` and
`.gitignore` files, skipping binary builds and compiled byte-codes (which you
usually don't want to touch anyway).

This is my reason to use it - less latency doing task I'm doing often.

## Installation

Binary builds:

 - [Linux 64 bit](http://solovyov.net/files/gr-latest-64-linux)
 - [Linux 32 bit](http://solovyov.net/files/gr-latest-32-linux)
 - [OS X 64 bit](http://solovyov.net/files/gr-latest-64-osx)
 - [OS X 32 bit](http://solovyov.net/files/gr-latest-32-osx)
 - [Windows 64 bit](http://solovyov.net/files/gr-latest-64-win.exe)
 - [Windows 32 bit](http://solovyov.net/files/gr-latest-32-win.exe)

It's suited to be installed via `go` tool, so you can do usual thing:

    go get github.com/piranha/goreplace

And you should be done. You have to have `$GOPATH` set for this to work (`go`
will put sources and generated binary there). Add `-u` flag there to update your
gr.

I prefer name `gr` to `goreplace`, so I link `gr` somewhere in my path (usually
in `~/bin`) to `$GOPATH/bin/goreplace`. **NOTE**: if you use `oh-my-zsh`, it
aliases `gr` to `git remote`, so you either should use another name (I propose
`gor`) or remove `gr` alias:

```
mkdir -p ~/.oh-my-zsh/custom && echo "unalias gr" >> ~/.oh-my-zsh/custom/goreplace.zsh
```

## Usage

Usage is pretty simple, you can just run `gr` to see help on options. Basically
you just supply a regexp (or a simple string - it's a regexp always as well) as
an argument and gr will search for it in all files starting from the
current directory, just like this:

    gr somestring

Some directories and files can be ignored by default (`gr` is looking for your
`.hgignore`/`.gitignore` in parent directories), just run `gr` without any
arguments to see help message - it contains information about them.

And to replace:

    gr somestring -r replacement.

It's performed in place and no backups are made (not that you need them, right?
You're using version control, aren't you?). Unfortunately only plain strings are
supported as replacement, no regexp submatch support yet (planned, though).

## Changelog

 - 0.5.0
   - `-o`/`--only` option  - include only files specified (thanks to Vignesh
     Sarma)
   - fixed support of recursive `*` in `.gitignore`
 - 0.4.3
   - make ignorers cross-platform (fixed them for windows)
 - 0.4.2
   - cleanup error reporting
 - 0.4.1
   - hide non-fatal errors by default
 - 0.4.0
   - option for plain-text searching
