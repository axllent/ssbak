# Changelog

## [1.1.7]

- Remove deprecated use of io/ioutil, resolves uncommon error of `archive/tar: write too long`
- Update Go dependencies, update minimum Go version, downgrade go-mysqldump


## [1.1.6]

- Exclude more resampled files, including converted (webp) and FocusPoint ones when using `--ignore-resampled`
- Workaround for Silverstripe 5 thumbnails when using `--ignore-resampled`
- Update Go dependencies


## [1.1.5]

- Update go-mysqldump (adds support for dumping VIEW tables)


## [1.1.4]

- Update Go modules
- Update GitHub workflows


## [1.1.3]

- Prevent errors on import due to `STRICT_TRANS_TABLES` and `STRICT_ALL_TABLES`


## [1.1.2]

- Allow .env loading to be skipped, and overwriting variables via environment (@ZaneA #6)
- Update Go libs


## [1.1.1]

- Fix self-updater (sorry!). If you are running 1.1.0 the self-updater will not work, so you will have to update manually unfortunately.
- Update Go libraries + security updates


## [1.1.0]

- Native support for importing and exporting databases, eliminating MySQL client/server version incompatibility issues


## [1.0.3]

- Revert `--set-gtid-purged=OFF` to dump arguments to prevent GTIDs warning - not supported on all systems
- Silently ignore GTIDs warnings


## [1.0.2]

- Add `--set-gtid-purged=OFF` to dump arguments to prevent GTIDs warning
- Update go modules


## [1.0.1]

- Revert to adding password to mysqldump arguments (older MySQL clients do not support env variable)


## [1.0.0]

- Add GitHub Actions to build binaries
- Add `--column-statistics=0` for supported MySQL (v8) servers


## [0.1.4]

- Disable pre-release versions for update notifications / updates


## [0.1.4-beta]

- Add support for `.env` and `assets` symbolic links


## [0.1.3]

- Audit with [gosec](https://github.com/securego/gosec) (Golang Security Checker) and security fixes


## [0.1.2]

- Add support for `SS_DATABASE_PREFIX`, `SS_DATABASE_SUFFIX` & `SS_DATABASE_CHOOSE_NAME`


## [0.1.1]

- Bugfix: Set correct path when restoring assets to set project root


## [0.1.0]

- Remove darwin 32-bit builds - no longer supported by Go 1.15
- Support for configs in parent directory


## [0.0.9-beta]

- Add `--no-tablespaces` to mysqldump command to prevent MySQL user permission errors in some cases.
- Pass mysql(dump) passwords on the command line. MySQL v5.7 does not support `MYSQL_PWD`, `--defaults-file` overrides system settings, and `--defaults-extra-file` is superseded by the existence of `~/.my.cnf`. This seems to be the only reliable solution for all versions of MySQL/MariaDB.


## [0.0.8-beta]

- Ensure correct path is used when detecting available space (*nix)
- Extract directories with 0755 permissions until chmod()


## [0.0.7-beta]

- Switch to compressed releases (tar.gz/zip). This release will have both compressed and uncompressed to allow older versions to catch up.


## [0.0.6-beta]

- Add drive checks to ensure enough space for sspak creation / extraction


## [0.0.5-beta]

- Update directory permissions after extraction completes
- Return errors if Cleanup() fails
- Alias `--ignore-resampled` with `-i`


## [0.0.4-beta]

- Save file & directory permissions, timestamps, uid & gid
- Restore file & directory permissions, timestamps, uid & gid (if permitted)
- Add `--ignore-resampled` option to save or restore without resampled images


## [0.0.3-beta]

- Add shell completion generator (see `ssbak completion -h`)
- Rename some core functions
- Support for MySQL port setting
- Allow exported environment values to override `.env` / `_ss_environment.php` values
- Add DB type / function map, and return errors for non-supported database types


## [0.0.2-alpha]

- Ignore PHP comments in `_ss_environment.php`


## [0.0.1-alpha]

- Initial alpha release
