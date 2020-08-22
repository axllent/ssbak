# Changelog

## [develop]

permitted- Save file & directory permissions, timestamps, uid & gid
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
