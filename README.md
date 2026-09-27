# Archive

A small CLI to save and search web pages and documents in a local SQLite database (with FTS5 full-text search).

## Usage

```sh
archive add <url | path/to/file>   # save a page or file (.txt, .md, .pdf, .doc, .docx, .odt, .odf)
archive list                       # list saved items
archive search 'terms'             # full-text search
archive del <id>                   # delete an item
```

## Build

```sh
go build -o archive
```

## Roadmap

- A desktop frontend is on the way.
