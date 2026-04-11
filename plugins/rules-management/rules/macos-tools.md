# macOS GNU Tool Replacements

On macOS, BSD versions of core tools lack GNU features. Use GNU versions installed via Homebrew:

- `ggrep` instead of `grep` (for `-P` Perl regex, `-oE` extended patterns, etc.)
- `gawk` instead of `awk` (for GNU awk extensions)
- `gsed` instead of `sed` (for GNU sed features)
- `gseq` instead of `seq` (macOS `seq` formats large integers in scientific notation)
- `gdate` instead of `date` (for `-d` date parsing, `--date`, format flags)

## Install

```bash
brew install grep gawk gnu-sed coreutils
```

## Usage

If `grep` fails or gives unexpected results on macOS, switch to `ggrep`.
If `awk` fails with syntax errors on macOS, switch to `gawk`.
If `sed` fails on macOS, switch to `gsed`.
If `seq` outputs scientific notation for large numbers, switch to `gseq`.
If `date` fails with `-d` or `--date` flags on macOS, switch to `gdate`.
