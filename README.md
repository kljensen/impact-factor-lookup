
## Impact factor lookup

Find the impact factors of the journals in which articles are published.
Expects article lists in XML format from Scopus. And expects the
`all.csv` file from
[Michael-E-Rose/SCImagoJournalRankIndicators](https://github.com/Michael-E-Rose/SCImagoJournalRankIndicators).

## Building

Just do `go build` in the directory.

## Running

```sh
./impact-factor-lookup \
    ~/data/export-102424.xml \
    ~/src/github.com/Michael-E-Rose/SCImagoJournalRankIndicators/all.csv \
    >sorted-papers.bib
```

Papers are output in descending order of impact factor. The latest impact
factor available for each journal is used.
