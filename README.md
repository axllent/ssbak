# SSBak - sspak database/asset backup & restore tool for Silverstripe

[![Go Report Card](https://goreportcard.com/badge/github.com/axllent/ssbak)](https://goreportcard.com/report/github.com/axllent/ssbak)


**SSBak** is a backup & restore tool for [Silverstripe](https://www.silverstripe.org) websites, written in Go. It backs up the assets and database, and is heavily based on (and largely compatible with) [SSPak](https://github.com/silverstripe/sspak).

### Why rewrite SSPak?

It was written to solve the backup/restore size limitations of the original SSPak utility, and can largely work as a drop-in replacement for SSPak (see [features](#features) and [limitations](#limitations)). SSPak has a nasty file size limitation due to undocumented PharData limits (see [1](https://github.com/silverstripe/sspak/issues/53), [2](https://github.com/silverstripe/sspak/issues/29) and [3](https://github.com/silverstripe/sspak/pull/52)). I have personally experienced these issues with SSPak and archives over 4GB, which resulted in partial asset backups with no warnings or errors at the time of backup. 

**SSBak does not have these file size limitations**.


## Features

- Compatible with the default `*.sspak` file format (tar non-executable files).
- Create and restore database and/or assets from a Silverstripe website regardless of asset / database size.
- Optionally create or restore without resampled images (`--ignore-resampled`). Note: this skips most common image manipulations except for `ResizedImages` which are usually generated for HTMLText and cannot be regenerated "on the fly".
- SSBak does not require (or use) PHP at all (see [limitations](#limitations)).
- Multiplatform static binaries (Linux, Mac & Windows). The only system requirements are `mysql`(.exe) and `mysqldump`(.exe) in your path. All other actions such as tar, gzip etc are handled directly in SSBak.
- Checks temporary and output locations have sufficient storage space **before** doing operations (Linux / Mac only)
- Optional verbose output to see what it is doing.
- Shell completion (see `ssbak completion -h`)


## Usage

```
SSBak - sspak database/asset backup & restore tool for Silverstripe.

Support/Documentation
  https://github.com/axllent/ssbak

Usage:
  ssbak [command]

Available Commands:
  extract      Extract .sspak backup
  load         Restore database and/or assets from .sspak backup
  save         Create .sspak backup of database and/or assets
  saveexisting Create .sspak backup from existing database SQL dump and/or assets
  version      Display the app version & update information

Flags:
  -h, --help   help for ssbak

Use "ssbak [command] --help" for more information about a command.
```


## Installation & requirements

- Download a suitable binary for your architecture (see [releases](https://github.com/axllent/ssbak/releases/latest)), extract the make it executable and place it in your $PATH. You can optionally save this as SSPak to use as a drop-in replacement for SSPak (see [limitations](#limitations)).
- MySQL and MySQLDump must be installed and in your $PATH. SSBak uses these system tools for backing up and restoring database backups.

To compile SSBak from source: `go install github.com/axllent/ssbak` (Go >= 1.14 required).


## Environment settings

SSBak automatically tries to parse either a `.env` or a `_ss_environment.php` in your webroot to detect the database settings. You can however export (or override) any of the following values by exporting them first in your shell:

- `SS_DATABASE_SERVER` **(required)**
- `SS_DATABASE_NAME` **(required)** (supports `SS_DATABASE_PREFIX`, `SS_DATABASE_SUFFIX` & `SS_DATABASE_CHOOSE_NAME`)
- `SS_DATABASE_USERNAME` **(required)**
- `SS_DATABASE_PASSWORD`
- `SS_DATABASE_PORT`
- `SS_DATABASE_CLASS` (currently only MySQL supported & defaults to MySQL if unspecified)


By default SSBak uses your system temporary directory (eg: `/tmp/` on Linux/Mac) to save and load the temporary files from your .sspak archive. You can override this path by setting the `TMPDIR` in your command:

```
TMPDIR="/drive/with/more/space" ssbak save . website.sspak
```


## Limitations

SSBak is designed as a database & asset backup & restore tool, and is largely drop-in replacement for the existing SSPak tool. There are however a few limitations:

- SSBak currently only supports MySQL databases. If there is demand for PostgreSQL then this can be requested and may be added in the future.
- SSBak is written in Go which does not have any PHP-parsing capabilities (it uses regular expressions). For all database dump & restore operations it requires either a `.env` or a `_ss_environment.php` file containing `SS_DATABASE_SERVER`, `SS_DATABASE_USERNAME`, `SS_DATABASE_PASSWORD` & `SS_DATABASE_NAME` in the **root** or parent directory of your website folder. You can however also export the required variables (see [Environment settings](#environment-settings)).
- It does not (yet?) support remote ssh storage, `git-remote` / `install`, or CSV import/export features from SSPak.


## Issues & vulnerabilities

Issues and vulnerabilities should be reported via the [Github issues tracker](https://github.com/axllent/ssbak/issues).


## Contributing

Code contributions should be supplied in the form of a merge request, and forked from the `develop` branch.
