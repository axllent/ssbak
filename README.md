# SSBak - asset and database backup tool for SilverStripe

[![Go Report Card](https://goreportcard.com/badge/github.com/axllent/ssbak)](https://goreportcard.com/report/github.com/axllent/ssbak)


**SSBak** is a backup & restore tool for SilverStripe websites written in Go. It backs up the assets and database, and based on (and largely compatible with) [SSPak](https://github.com/silverstripe/sspak).

### Why rewrite SSPak?

It was written to address backup/restore size limitations of the original SSPak utility, and can largely work as a drop-in replacement for SSPak (see [features](#features) and [limitations](#limitations)). SSPak has a nasty file size limitation due to undocumented PharData limits (see [1](https://github.com/silverstripe/sspak/issues/53), [2](https://github.com/silverstripe/sspak/issues/29) and [3](https://github.com/silverstripe/sspak/pull/52)). I have personally experienced these issues with SSPak and archives over 4GB, which resulted in partial assets backups with no warnings or errors at the time of backup. 

**SSBak does not have these file size limitations**.


## Features

- Completely compatible with the default .sspak file format (tar non-executable files).
- Create and restore database and/or assets from a SilverStripe website regardless of asset / database size.
- Create or restore without resampled images (`--ignore-resampled`). Note: this skips most common image manipulations except for ResizedImages which are usually generated for HTMLText and cannot be regenerated "on the fly". Experimental.
- Does not require (or use) PHP (see [limitations](#limitations)).
- Multiplatform static binaries (Linux, Mac & Windows). The only system requirements are `mysql`(.exe) and `mysqldump`(.exe). All other actions such as tar, gzip etc are handled directly in SSBak.
- Optional verbose output to see what it is doing.
- Shell completion (see `ssbak completion -h`)


## Usage

```
SSBak - sspak database/asset backup tool for SilverStripe.

Usage:
  ssbak [command]

Available Commands:
  extract      Extract an .sspak archive
  help         Help about any command
  load         Restore an .sspak backup of your database and/or assets
  save         Create an .sspak backup of your database and/or assets
  saveexisting Create an .sspak file from an existing database SQL dump and/or assets folder
  version      Display the app version & update information

Flags:
  -h, --help   help for ssbak

Use "ssbak [command] --help" for more information about a command.
```


## Installation & requirements

- Download a suitable binary for your architecture (see [releases](https://github.com/axllent/ssbak/releases/latest)), make it executable and place it in your $PATH. You can optionally save this as SSPak to use as a drop-in replacement for SSPak (see [limitations](#limitations)).
- MySQL and MySQLDump must be installed. SSBak uses these system tools for backing up and restoring database backups.

If you wish to compile SSBak from source you can `go get -u github.com/axllent/ssbak` (Go >= 1.11 required).


## Environment settings

SSBak automatically tries to parse either a `.env` or a `_ss_environment.php` in your webroot to detect the database settings. You can however export (or override) any of the following by exporting them first in your shell:

- `SS_DATABASE_SERVER` **(required)**
- `SS_DATABASE_NAME` **(required)**
- `SS_DATABASE_USERNAME` **(required)**
- `SS_DATABASE_PASSWORD`
- `SS_DATABASE_PORT`
- `SS_DATABASE_CLASS` (currently only mysql supported & defaults to MySQL if unspecified)


By default SSBak uses your system temporary directory (eg: `/tmp/` on Linux/Mac) to save and load the temporary files from your .sspak archive. You can override this path by setting the `TMPDIR` in your command:

```
TMPDIR="/drive/with/more/space" ssbak save . website.sspak
```


## Limitations

SSBak is designed as a database & asset backup & restore tool, and is largely drop-in replacement for the existing SSPak tool. There are however a few limitations:

- SSBak only supports MySQL databases. If there is demand for PostgreSQL then this can be requested and may be added in the future.
- SSBak is written in Go which does not have any PHP-parsing capabilities. For all database dump & restore operations it requires either a `.env` or a `_ss_environment.php` file containing `SS_DATABASE_SERVER`, `SS_DATABASE_USERNAME`, `SS_DATABASE_PASSWORD` & `SS_DATABASE_NAME` in the **root** of your website folder (default location). You can however also export the required variables (see [Environment settings](#environment-settings))
- It does not (yet?) support remote ssh storage, `git-remote` / `install`, or CSV import/export features from SSPak.
