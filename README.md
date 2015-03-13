unzipall
========

Unzip all zip files in `src` directory to `dst` directory.

## Install

```
go install github.com/gedex/unzipall
```

## Instruction

Suppose you have directory `mydir`:

```
tree mydir
mydir/
├── sub
│   ├── 1.zip
│   └── 2.zip
└── README

1 directory, 3 files
```

Running `unzipall` with:

```
unzipall -src=./mydir/ -dst=./mydir/
```

will result in:

```
tree mydir
mydir/
README
└── sub
    ├── 1
    │   └── a.bin
    ├── 1.zip
    ├── 2
    │   └── b.bin
    └── 2.zip

3 directories, 5 files
```
