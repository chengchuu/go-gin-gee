## 背景

最近面对一个庞大的项目，但是只需要修改某个模块，每次都手搓命令太麻烦了，于是就想着能不能写个脚本来辅助处理这些事情。

## 解决方案

定制化一键 ESLint，执行文件下载地址：

<https://github.com/chengchuu/go-gin-gee/releases/tag/v1.4.0>

![Assets](http://blog.mazey.net/wp-content/uploads/2023/10/assets-Releasev1.4.0-mazeyqian_go-gin-gee.png)

### 基础使用

以下案例以 MacOS 为例，其他系统自行替换对应的文件。

案例 1：指定文件 `file1.js` 和 `file2.js`，使用默认的配置。

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 -files="file1.js,file2.js"
```

案例 2：指定文件夹 `src/views` 和 `src/components`。

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 -folders="/root/app/src/views,/root/app/src/components"
```

配合根目录 `root` 使用指定文件夹：

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 \
  -folders="src/views,src/components" \
  -root="/root/app/"
```

案例 3：指定 ESLint 配置文件 `custom.eslintrc.js` 和命令 `--fix`。

```bash
#!/bin/bash
./eslint-files-mac-darwin-amd64 \
  -folders="/root/app/src/views" \
  -esConf="custom.eslintrc.js" \
  -esCom="--fix"
```

### 复杂场景

1. 指定 ESLint 配置文件 `custom.eslintrc.js`；
2. 指定附带命令 `--fix`；
3. 指定文件和文件夹；
4. 指定文件后缀；
5. 添加前置和后置执行命令。

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

### 参数说明

| 参数 | 说明 | 默认 | 示例 | 是否必须 |
| --- | --- | --- | --- | --- |
| `files` | 指定文件，多个文件用 `,` 分隔。 | - | `file1.js,file2.js` | 可选 |
| `folders` | 指定文件夹，多个文件夹用 `,` 分隔。 | - | `src/views,src/components` | 可选 |
| `esConf` | 指定 ESLint 配置文件。 | - | `custom.eslintrc.js` | 可选 |
| `esCom` | 指定附带命令。 | - | `--fix` | 可选 |
| `root` | 指定根目录，配合 `folders` 使用。 | - | `/root/app/` | 可选 |
| `ext` | 指定文件后缀。 | `.js` | `.js,.ts,.jsx,.vue` | 可选 |
| `befCom` | 指定前置执行命令。 | - | `echo 'Starting format';` | 可选 |
| `aftCom` | 指定后置执行命令。 | - | `echo 'Format completed';` | 可选 |
| `filesRang` | 指定文件范围，统计处理过和未处理的文件。 | - | `/root/app/` | 可选 |

### 演示效果

![ESLint Files](http://blog.mazey.net/wp-content/uploads/2023/10/eslint-Screen-Shot-w601.png)

## 项目地址

该脚本使用 Go 语言开发，访问地址：

<https://github.com/chengchuu/go-gin-gee/tree/main/scripts/eslint-files>

**版权声明**

本博客所有的原创文章，作者皆保留版权。转载必须包含本声明，保持本文完整，并以超链接形式注明作者[除除](https://github.com/chengchuu)和本文原始地址：<http://blog.mazey.net/4207.html>

（完）
