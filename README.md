# SSBak - asset and database backup tool for SilverStripe

**SSBak** is a backup & restore tool for SilverStripe websites, backing up the assets and database, based on and largely compatible with [SSPak](https://github.com/silverstripe/sspak). It backups up just the assets and database, not the website code itself.

It was written to address backup/restore size limitations of the original `sspak` utility, and can largely work as a drop-in replacement for `sspak` (see [features](#features) and [limitations](#limitations)). The original `sspak` PHP version has a nasty file size limitation due to undocumented PharData limits (see [1](https://github.com/silverstripe/sspak/issues/53), [2](https://github.com/silverstripe/sspak/issues/29) and [3](https://github.com/silverstripe/sspak/pull/52)). I have personally experienced these issues with `sspak` with archives over 4GB, resulting in partial assets backups with no warnings or errors at the time of backup. 

**SSBak does not have these file size limitations**.


## Features

- Completely compatible with existing .sspak file format.
- Create and restore database and/or assets from a SilverStripe website.
- Does not require PHP.
- Multi-platform static binaries (Linux, Mac & Windows), written in Go. The only system requirements are `mysql`(.exe) and `mysqldump`(.exe).
- No archive size limitations.
- Optional verbose output.


## Installation & requirements

- Download a suitable binary for your architecture (see [releases](https://github.com/axllent/ssbak/releases/latest)), make it executable and place it in your $PATH. You can optionally save this as `sspak` to use as a drop-in replacement for `sspak` (see [limitations](#limitations)).
- MySQL and MySQLDump must be installed. SSBak uses these system tools for backing up and restoring database backups.

If you wish to compile SSBak from source you can `go get -u github.com/axllent/sspak` (Go >= 1.11 required).


## Limitations

SSBak is designed as a database & asset backup & restore tool, and is largely drop-in replacement for the existing `sspak` tool. There are however a things to consider:

- SSBak currently only supports MySQL databases. If PostgreSQL is requested then this may be added.
- SSBak is written in Go, and does not have PHP-parsing capabilities. For all database dump & restore operations it requires either a `.env` or a `_ss_environment.php` file containing `SS_DATABASE_SERVER`, `SS_DATABASE_USERNAME`, `SS_DATABASE_PASSWORD` & `SS_DATABASE_NAME` in the **root** of your website folder.
- It does not support remote ssh storage sspak features.
- It does not support `git-remote` / `install` sspak features.


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

Flags:
  -h, --help   help for ssbak

Use "ssbak [command] --help" for more information about a command.
```

### Temporary directory

By default SSBak uses your system temporary directory (eg: `/tmp/` on Linux/Mac) to create, merge and extract .sspak files. You can override this path by setting the `TMPDIR` in your command:

```
TMPDIR="/drive/with/more/space" ssbak save . website.sspak
```
