# Changelog

## [0.0.6-beta]

- Add drive checks to ensure enough space for sspak creation / extraction


## [0.0.5-beta]

- Update directory permissions after extraction completes
- Return errors if Cleanup() fails
- Alias `--ignore-resampled` with `-i`


## [0.0.4-beta]

- Save file & directory permissions, timestamps, uid & gid
- Restore file & directory permissions, timestamps, uid & gid (if permitted)
- Add `--ignore-resampled` option to save or restore without resampled images.


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
