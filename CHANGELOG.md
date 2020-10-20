# Changelog

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
