# Wget

## Description:

This project is a custom implementation of the wget tool in Go. It is designed to provide various functionalities for downloading files and mirroring websites. Key features include support for downloading in the background, setting rate limits, mirroring websites, and handling multiple download links from a file. The tool offers flexibility through a variety of command-line flags.

## Features:

- Background Downloading: Download files in the background using the -B flag.
- Output Customization: Specify output file name with -O and directory path with -P.
- Rate Limiting: Control download speed using the --rate-limit option.
- Input File Processing: Download multiple links provided in a file using -i.
- Website Mirroring: Mirror entire websites with the --mirror flag.
- File Type Rejection: Reject specific file types during download using -R.
- Directory Exclusion: Exclude specific directories from downloads with -X.

## Usage:

### Basic Download:
```
go run main.go [URL]
```
### Download with Custom Filename:

````
go run main.go -O [filename] [URL]
``````
### Background Download:

```
go run main.go -B [URL]
```
### Rate Limited Download:

```
go run main.go --rate-limit [speed] [URL]
```
### Download From Input File:

```
go run main.go -i [inputfile]
```
### Mirror a Website:

```
go run main.go --mirror [URL]
```
### Reject Specific File Types:

```
go run main.go -R [filetypes] [URL]
```
### Exclude Directories:

```
go run main.go -X [directories] [URL]
```

Enjoy :=)