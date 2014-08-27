# Go Replace

Go Replace (gr) is a simple utility which can be used as replacement for grep +
sed combination in one of most popular cases - find files, which contain
something, possibly replace this with something else. Main points:

 - Reads `.hgignore`/`.gitignore` to skip files
 - Skips binaries
 - Familiar PCRE-like regexp syntax
 - Can perform replacements
 - Fast

Bonus:

 - Can search in file names with `-f` (i.e. a simple alternative to `find`)

[![Build Status](https://travis-ci.org/piranha/goreplace.png)](https://travis-ci.org/piranha/goreplace)

[Releases and changelog](https://github.com/piranha/goreplace/releases)

## Why

Why do thing which is done by grep, find, and sed? Well, for one - I grew tired
of typing long commands with pipes and ugly syntax. You want to search? Use
grep. Replace? Use find and sed! Different syntax, context switching,
etc. Switching from searching to replacing with gr is 'up one item in
history and add a replacement string', much simpler!

Besides, it's also faster than grep! Hard to believe, and it's a bit of cheating -
but gr by default ignores everything you have in your `.hgignore` and
`.gitignore` files, skipping binary files and compiled bytecodes (which you
usually don't want to touch anyway).

This is my reason to use it - less latency doing task I'm doing often.

## Installation

Just download a suitable binary from
[release page](https://github.com/piranha/goreplace/releases). Put this file in
your `$PATH` and rename it to `gr` to have easier access.

### Building from source

You can also install it from source, if that's your thing:

    go get github.com/piranha/goreplace

And you should be done. You have to have `$GOPATH` set for this to work (`go`
will put sources and generated binary there). Add `-u` flag there to update your
`gr`.

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
