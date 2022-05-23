# agenex

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE.md)

Script to convert .agenda (apple [agenda](https://www.agenda.com/) notebook export) to .enex ([evernote](https://evernote.com/) notebook export)

## Supported objects & styles

Item|Converted
|--|--|
Text|✅
List|✅
**Embedded objects**|
0: Hashtag|Plain text with # prefix
1: Mention|Plain text with @ prefix
2:|Plain text
3:|Plain text
4:|Plain text
5: Hyperlink|✅
6: Agenda internal link to other note|❌
7: Attachment|✅
8:|Plain text
9: Action list|Plain text

## Specs

### .agenda

Zip archive with following structure

```
.agenda
├── Archive
│   ├── Attachments
│   │   ├── *
│   ├── Data.json
```

### .enex
 
XML format. Official sample [here](https://gist.github.com/evernotegists/6116886)

## Development

### Generate extension-to-mime mappings

Run `make gen` to generate extension-to-mime mappings into `mmap.go` to be used in main module.

Alternatively,

```sh
go run ./mmap ./mmap/mime.types > mmap.go
go fmt .
```

### Build binaries

Run `make all` to build binaries (see [Makefile](Makefile))

## Usage

Run `./bin/agenex input_folder output_folder` to convert all `.agenda` in input_folder to `.enex` in output_folder
