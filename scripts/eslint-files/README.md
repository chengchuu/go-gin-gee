# Using Scripts to Consolidate Designated Files/Folders and Execute Customized ESLint Commands

## Background

Recently, I faced a large project where I only needed to modify a specific module. It was too cumbersome to manually input commands every time, so I thought about writing a script to assist in handling these tasks.

## Solution

Customized one-click ESLint, download the executable file at:

<https://github.com/chengchuu/go-gin-gee/releases/tag/v1.4.0>

![Assets](http://blog.mazey.net/wp-content/uploads/2023/10/assets-Releasev1.4.0-mazeyqian_go-gin-gee.png)

### Basic Usage

The following examples use MacOS as an example, please replace the corresponding files for other systems.

Example 1: Specify files `file1.js` and `file2.js` with the default configuration.

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 -files="file1.js,file2.js"
```

Example 2: Specify folders `src/views` and `src/components`.

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 -folders="/root/app/src/views,/root/app/src/components"
```

Specify folders using the root directory `root`:

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 \
  -folders="src/views,src/components" \
  -root="/root/app/"
```

Example 3: Specify ESLint configuration file `custom.eslintrc.js` and command `--fix`.

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 \
  -folders="/root/app/src/views" \
  -esConf="custom.eslintrc.js" \
  -esCom="--fix"
```

### Complex Scenarios

1. Specify ESLint configuration file `custom.eslintrc.js`;
2. Specify accompanying command `--fix`;
3. Specify files and folders;
4. Specify file suffix;
5. Add prefix and postfix execution commands.

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 \
  -files="file1.js,file2.js" \
  -folders="src/views,src/components" \
  -root="/root/app/" \
  -esConf="custom.eslintrc.js" \
  -esCom="--fix" \
  -ext=".js,.ts,.jsx,.vue,.tsx" \
  -befCom="echo 'Starting format';" \
  -aftCom="echo 'Format completed';"
```

### Parameter Description

| Parameter | Description | Default | Example | Required |
| --- | --- | --- | --- | --- |
| `files` | Specify files, multiple files are separated by `,`. | - | `file1.js,file2.js` | Optional |
| `folders` | Specify folders, multiple folders are separated by `,`. | - | `src/views,src/components` | Optional |
| `esConf` | Specify ESLint configuration file. | - | `custom.eslintrc.js` | Optional |
| `esCom` | Specify accompanying command. | - | `--fix` | Optional |
| `root` | Specify root directory, used with `folders`. | - | `/root/app/` | Optional |
| `ext` | Specify file suffix. | `.js` | `.js,.ts,.jsx,.vue` | Optional |
| `befCom` | Specify prefix execution command. | - | `echo 'Starting format';` | Optional |
| `aftCom` | Specify postfix execution command. | - | `echo 'Format completed';` | Optional |
| `filesRang` | Specify the range of files, count the processed and unprocessed files. | - | `/root/app/` | Optional |
