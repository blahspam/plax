# Plax: Plex Local Asset eXporter.

Utility for exporting posters and backgrounds from `Plex`.

## Installation

```bash
$ go install github.com/blahspam/plax 
```

## Usage

```bash
$ plax -u http://localhost:32400 -t $PLEX_TOKEN
```

### Arguments

| flag           | description                               | default value          | environment variable |
|----------------|-------------------------------------------|------------------------|----------------------|
| --url (-u)     | Plex base URL                             | http://localhost:32400 | `$PLEX_HOST`         |
| --token (-t)   | Plex authentication token                 |                        | `$PLEX_TOKEN`        |
| --quiet (-q)   | less verbose output                       | false                  |                      |  
| --output-dir   | alternative directory for exported assets |                        |                      |
| --help (-h)    | show help                                 |                        |                      |
| --version (-v) | print version                             |                        |                      |

*Note* `--output-dir` may be used to test the export by writing results to a 
temporary location for review.

## Assumptions

This utility assumes your Plex library is structured according to current best 
practices including:

* [Movies in Their Own Folder](https://support.plex.tv/articles/naming-and-organizing-your-movie-media-files/#toc-0)
* [Standard, Season-Based Shows](https://support.plex.tv/articles/naming-and-organizing-your-tv-show-files/#toc-0)
