# ssg is yet another static site generator

For a real world usage example see [klingtnet/klingtnet.github.io](https://github.com/klingtnet/klingtnet.github.io).

## Installation

If you've a Go distribution installed then just run:

```sh
$ go install github.com/klingtnet/static-site-generator/cmd/ssg@latest
```

Otherwise you can use one of the pre build binaries from the [releases](https://github.com/klingtnet/static-site-generator/releases) page.

## Usage

First you need a configuration file, for all available options refer to [`example.config.json`](https://github.com/klingtnet/static-site-generator/blob/master/config.example.json).
Second, and most important, is content.  The absolute minimum is a folder containing just an `index.md`.  The folder structure of a more complex page is shown below:

```
content/
├── about-me.md
├── articles
│   ├── bye.md
│   └── hello.md
├── index.md
├── images
    └── photo.webp
└── notes
    └── index.md
    └── something-else.md
static/
└── assets
    └── base.css
```

For this example `ssg` will generate:

- a navigation menu containing all root pages and first-level subdirectories with markdown files, i.e. `[home,about,articles,notes]`
- a list page for the `articles` directory (since no `index.md` was present)
- no list page for `notes`, instead `index.md` is assumed to be the list page
- `images` is just copied
- contents from `static` directory will copied as is

```
output/
├── about-me.html
├── articles
│   ├── bye.html
│   ├── hello.html
│   └── index.html
├── index.html
└── notes
    ├── index.html
    └── something-else.html
└── assets
    └── base.css
```

Anything besides the root `index.md` is optional.