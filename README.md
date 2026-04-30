# go-font

Go 语言 TrueType 字体（.ttf）解析、编辑与序列化库，支持 WOFF / WOFF2 / EOT 格式。

## 安装

```bash
go get github.com/venusliang/go-font
```

## 快速开始

```go
package main

import (
    "fmt"
    "os"

    gofont "github.com/venusliang/go-font"
)

func main() {
    // 读取字体文件
    data, _ := os.ReadFile("myfont.ttf")

    // 解析
    ttf, err := gofont.Parse(data)
    if err != nil {
        panic(err)
    }

    // 查看字体信息
    fmt.Printf("字形数量: %d\n", ttf.NumGlyphs())

    // 序列化回写
    out, _ := ttf.Serialize()
    os.WriteFile("output.ttf", out, 0644)
}
```

## API 概览

### 解析与序列化

| 方法 | 说明 |
|------|------|
| `Parse(data []byte) (TrueTypeFont, error)` | 解析 TTF 二进制数据，返回字体对象 |
| `ParseWOFF(data []byte) (TrueTypeFont, error)` | 解析 WOFF 二进制数据，返回字体对象 |
| `ParseWOFF2(data []byte) (TrueTypeFont, error)` | 解析 WOFF2 二进制数据，返回字体对象 |
| `ParseEOT(data []byte) (TrueTypeFont, error)` | 解析 EOT 二进制数据，返回字体对象 |
| `ttf.Serialize() ([]byte, error)` | 将字体对象序列化为完整的 TTF 二进制数据 |
| `ttf.SerializeWOFF() ([]byte, error)` | 将字体对象序列化为 WOFF 格式 |
| `ttf.SerializeWOFF2() ([]byte, error)` | 将字体对象序列化为 WOFF2 格式 |
| `ttf.SerializeEOT() ([]byte, error)` | 将字体对象序列化为 EOT 格式 |

### Unicode 映射

| 方法 | 说明 |
|------|------|
| `RuneToGlyphID(r rune) uint16` | 根据Unicode码点查询字形ID，未映射返回0 |
| `GlyphForRune(r rune) *Glyph` | 根据Unicode码点获取字形数据，未映射返回nil |
| `SetRuneMapping(r rune, glyphID uint16) error` | 设置Unicode码点到字形ID的映射 |
| `RemoveRuneMapping(r rune)` | 删除指定Unicode码点的映射 |
| `SetRuneMappings(m map[rune]uint16) error` | 批量设置映射 |
| `RuneMappings() []struct{Rune; GlyphID}` | 返回所有映射，按Unicode码点排序 |
| `MappedRunes() []rune` | 返回所有已映射的Unicode码点 |

### 字形操作

| 方法 | 说明 |
|------|------|
| `NumGlyphs() int` | 返回字形总数 |
| `GlyphAt(index int) *Glyph` | 按索引获取字形，越界返回nil |
| `SetGlyphAt(index int, g *Glyph) error` | 替换指定索引的字形数据 |
| `AppendGlyph(g *Glyph) (int, error)` | 追加新字形，返回新索引 |
| `CopyGlyph(src, dst int) error` | 复制字形数据 |
| `RemoveGlyphs(indices []int) (remap, error)` | 删除指定索引的字形并压缩相关表 |
| `TranslateGlyph(index, dx, dy int16) error` | 平移字形坐标 |
| `ScaleGlyph(index int, sx, sy float64) error` | 缩放字形坐标 |

### 字形属性查询

| 方法 | 说明 |
|------|------|
| `IsSimpleGlyph(index int) bool` | 是否为简单字形 |
| `IsCompositeGlyph(index int) bool` | 是否为复合字形 |
| `GlyphBBox(index) (xMin, yMin, xMax, yMax, ok)` | 字形包围盒 |
| `PointCount(index int) int` | 字形点数 |
| `ContourCount(index int) int` | 字形轮廓数 |

### 字体度量

| 方法 | 说明 |
|------|------|
| `UnitsPerEm() uint16` | 设计空间单位 |
| `FontBBox() (xMin, yMin, xMax, yMax)` | 全局包围盒 |
| `Ascent() int16` | 上升量 |
| `Descent() int16` | 下降量 |
| `AdvanceWidth(glyphID uint16) uint16` | 字形前进宽度 |
| `AdvanceWidthForRune(r rune) uint16` | 按Unicode查询前进宽度 |
| `LeftSideBearing(glyphID uint16) int16` | 左侧间距 |
| `SetAdvanceWidth(glyphID, width uint16) error` | 设置前进宽度 |
| `SetLeftSideBearing(glyphID uint16, lsb int16) error` | 设置左侧间距 |

### 字体名称

| 方法 | 说明 |
|------|------|
| `FontFamily() string` | 获取字体族名 |
| `FontFullName() string` | 获取字体全名 |
| `SetFontFamily(name)` | 设置字体族名 |
| `SetFontFullName(name)` | 设置字体全名 |

### 多格式支持

库支持 TTF、WOFF、WOFF2、EOT 四种格式的解析与序列化，所有格式解析后均返回统一的 `TrueTypeFont` 对象，可自由编辑后再序列化为任意格式。

#### WOFF（Web Open Font Format）

```go
data, _ := os.ReadFile("myfont.woff")
ttf, _ := gofont.ParseWOFF(data)

// 编辑...
ttf.SetFontFamily("NewName")

// 序列化为 WOFF
woffOut, _ := ttf.SerializeWOFF()
os.WriteFile("output.woff", woffOut, 0644)

// 也可序列化为 TTF 或 WOFF2
ttfOut, _ := ttf.Serialize()
woff2Out, _ := ttf.SerializeWOFF2()
```

- 基于 zlib 逐表压缩
- 解析后等同于标准 TTF，所有编辑 API 均可用
- 序列化时自动压缩每个表

#### WOFF2（Web Open Font Format 2）

```go
data, _ := os.ReadFile("myfont.woff2")
ttf, _ := gofont.ParseWOFF2(data)

// 序列化为 WOFF2
woff2Out, _ := ttf.SerializeWOFF2()
```

- 基于 Brotli 压缩（所有表合并为单个压缩流）
- 自动处理 glyf/loca 和 hmtx 表的逆变换
- 序列化时不做表变换（transform version 3），兼容性好
- 依赖 `github.com/andybalholm/brotli`（纯 Go，无 CGO）

#### EOT（Embedded OpenType）

```go
data, _ := os.ReadFile("myfont.eot")
ttf, _ := gofont.ParseEOT(data)

// 序列化为 EOT
eotOut, _ := ttf.SerializeEOT()
```

- 微软字体嵌入格式，主要用于旧版 IE
- 支持 version 0x00010000 / 0x00020001 / 0x00020002
- 支持自动 XOR 0x50 解密
- 序列化时从 OS/2、head、name 表自动填充 EOT 元数据（PANOSE、Weight、UnicodeRange 等）
- 名称字段使用 UTF-16LE 编码

#### 格式限制

| 格式 | 限制 |
|------|------|
| WOFF | 无 |
| WOFF2 | 序列化时不做 glyf/loca/hmtx 表变换，压缩率略低于官方工具 |
| EOT | 不支持 MTX（MicroType Express）压缩，遇到时返回错误；不支持 CFF（OpenType with CFF）字体 |
| EOT | 仅支持 TrueType 轮廓（glyf 表），不支持 CFF 轮廓 |

#### 格式互转

```go
// 读取 WOFF2，输出为 TTF
ttf, _ := gofont.ParseWOFF2(woff2Data)
ttfOut, _ := ttf.Serialize()

// 读取 TTF，输出为 EOT
ttf, _ := gofont.Parse(ttfData)
eotOut, _ := ttf.SerializeEOT()

// 读取 EOT，输出为 WOFF
ttf, _ := gofont.ParseEOT(eotData)
woffOut, _ := ttf.SerializeWOFF()
```

### 高级操作

| 方法 | 说明 |
|------|------|
| `Subset(keepRunes []rune) error` | 字体子集化，只保留指定字符需要的字形 |

## 使用示例

### 修改 Unicode 映射

将 Unicode 码点 0x91 的字形映射改为 0xFB：

```go
ttf, _ := gofont.Parse(data)

gid := ttf.RuneToGlyphID(0x91)   // 取出字形ID
ttf.RemoveRuneMapping(0x91)       // 删除旧映射
ttf.SetRuneMapping(0xFB, gid)     // 建立新映射

out, _ := ttf.Serialize()
```

### 查询字形信息

```go
ttf, _ := gofont.Parse(data)

// 通过Unicode码点查字形
g := ttf.GlyphForRune('A')
if g != nil {
    fmt.Printf("xMin=%d, yMin=%d, xMax=%d, yMax=%d\n",
        g.header.xMin, g.header.yMin,
        g.header.xMax, g.header.yMax)
}

// 遍历所有映射
for _, m := range ttf.RuneMappings() {
    fmt.Printf("U+%04X -> glyph %d\n", m.Rune, m.GlyphID)
}
```

### 裁剪字形表

删除不需要的字形，缩小字体文件体积：

```go
ttf, _ := gofont.Parse(data)

// 删除索引为 3, 7, 10 的字形
remap, err := ttf.RemoveGlyphs([]int{3, 7, 10})
if err != nil {
    panic(err)
}

// remap 记录了旧索引到新索引的映射关系
// 例如 remap[4] == 3 表示旧索引4变成了新索引3
// 被删除的索引不会出现在 remap 中

fmt.Printf("裁剪后字形数: %d\n", ttf.NumGlyphs())

out, _ := ttf.Serialize()
```

`RemoveGlyphs` 会自动处理以下关联数据：

- `glyf` — 删除对应字形，紧凑排列
- `loca` — 重新计算偏移量
- `hmtx` — 删除对应的水平度量数据
- `maxp` — 更新字形数量和统计信息
- `hhea` — 更新 numberOfHMetrics
- 复合字形 — 重新映射组件引用的旧字形索引
- cmap — 更新 Unicode 到字形ID的映射

> 注意：字形 0（.notdef）不可删除，它是字体必需的默认字形。

### 替换字形数据

```go
ttf, _ := gofont.Parse(data)

// 用索引0的字形替换索引1的字形
src := ttf.GlyphAt(0)
if src != nil {
    // 复制一份避免共享底层数据
    newGlyph := &gofont.Glyph{
        header: src.header,
    }
    if src.simpleGlyph != nil {
        sg := *src.simpleGlyph
        newGlyph.simpleGlyph = &sg
    }

    ttf.SetGlyphAt(1, newGlyph)
}

out, _ := ttf.Serialize()
```

### 添加新的 Unicode 映射

```go
ttf, _ := gofont.Parse(data)

// 将字符 'A' (U+0041) 映射到已有的字形1
err := ttf.SetRuneMapping('A', 1)
if err != nil {
    panic(err)
}

// 映射多个字符
chars := []rune{'A', 'B', 'C'}
for i, ch := range chars {
    ttf.SetRuneMapping(ch, uint16(i+1))
}

out, _ := ttf.Serialize()
```

### 读取字体基本信息

```go
ttf, _ := gofont.Parse(data)

fmt.Printf("字体名: %s\n", ttf.FontFamily())
fmt.Printf("全名: %s\n", ttf.FontFullName())
fmt.Printf("Units/Em: %d\n", ttf.UnitsPerEm())
fmt.Printf("Ascent: %d, Descent: %d\n", ttf.Ascent(), ttf.Descent())

// 查询字形度量和属性
w := ttf.AdvanceWidth(1)
lsb := ttf.LeftSideBearing(1)
xMin, yMin, xMax, yMax, _ := ttf.GlyphBBox(1)
pts := ttf.PointCount(1)
fmt.Printf("字形1: width=%d lsb=%d bbox=(%d,%d,%d,%d) points=%d\n",
    w, lsb, xMin, yMin, xMax, yMax, pts)

// 通过 Unicode 查询宽度
aw := ttf.AdvanceWidthForRune(0xE001)
fmt.Printf("U+E001 advance width: %d\n", aw)
```

### 字体子集化

```go
ttf, _ := gofont.Parse(data)

// 只保留需要的字符，自动删除多余字形
err := ttf.Subset([]rune{'A', 'B', 'C', 'D', 'E'})
if err != nil {
    panic(err)
}

fmt.Printf("子集化后字形数: %d\n", ttf.NumGlyphs())

out, _ := ttf.Serialize()
```

### 修改字体度量

```go
ttf, _ := gofont.Parse(data)

// 修改字形1的前进宽度
ttf.SetAdvanceWidth(1, 600)
ttf.SetLeftSideBearing(1, 20)

// 修改字体名称
ttf.SetFontFamily("MyFont")
ttf.SetFontFullName("MyFont Regular")

out, _ := ttf.Serialize()
```

### 字形几何变换

```go
ttf, _ := gofont.Parse(data)

// 平移字形 (向右100单位，向下50单位)
ttf.TranslateGlyph(1, 100, -50)

// 缩放字形 (X方向2倍，Y方向2倍)
ttf.ScaleGlyph(1, 2.0, 2.0)

out, _ := ttf.Serialize()
```

### 追加新字形

```go
ttf, _ := gofont.Parse(data)

// 创建一个空字形
newGlyph := &gofont.Glyph{
    header: gofont.GlyphHeader{
        numberOfContours: 1,
        xMin: 0, yMin: 0, xMax: 500, yMax: 700,
    },
    simpleGlyph: &gofont.SimpleGlyph{
        endPtsOfContours: []uint16{3},
        xCoordinates:     []int16{0, 500, 500, 0},
        yCoordinates:     []int16{0, 0, 700, 700},
    },
}

idx, _ := ttf.AppendGlyph(newGlyph)
ttf.SetRuneMapping('Z', uint16(idx))

out, _ := ttf.Serialize()
```

## 数据结构

### Glyph

```go
type Glyph struct {
    header         GlyphHeader      // 轮廓数、包围盒
    simpleGlyph    *SimpleGlyph     // 简单字形（非nil时为简单字形）
    compositeGlyph *CompositeGlyph  // 复合字形（非nil时为复合字形）
}
```

**GlyphHeader**

```go
type GlyphHeader struct {
    numberOfContours int16  // 轮廓数量（>=0为简单字形，<0为复合字形）
    xMin, yMin       int16  // 包围盒最小坐标
    xMax, yMax       int16  // 包围盒最大坐标
}
```

**SimpleGlyph**

```go
type SimpleGlyph struct {
    endPtsOfContours []uint16  // 每个轮廓的结束点索引
    instructions     []byte    // 提示指令
    flags            []uint8   // 每个点的标志位
    xCoordinates     []int16   // X坐标（绝对坐标）
    yCoordinates     []int16   // Y坐标（绝对坐标）
}
```

**CompositeGlyph**

```go
type CompositeGlyph struct {
    components []GlyphComponent  // 组件列表
}

type GlyphComponent struct {
    flags      uint16     // 组件标志
    glyphIndex uint16     // 引用的字形索引
    arg1, arg2 int16      // 位置参数
    transform  [4]int16   // 可选的2x2变换矩阵
}
```

## 支持的字体表

| 表 | 文件 | 说明 |
|----|------|------|
| `head` | `head.go` | 字体头，包含全局度量信息 |
| `hhea` | `hhea.go` | 水平布局头 |
| `hmtx` | `hmtx.go` | 水平度量（advance width + LSB） |
| `maxp` | `maxp.go` | 最大配置文件，字形数量 |
| `OS/2` | `os_2..go` | OS/2 度量信息 |
| `name` | `name.go` | 字体名称字符串 |
| `cmap` | `cmap.go` | 字符到字形映射（格式 0/4/6/12） |
| `loca` | `loca.go` | 字形索引到偏移映射 |
| `glyf` | `glyf.go` | 字形轮廓数据 |
| `post` | `post.go` | PostScript 名称映射 |
| `kern` | `kern.go` | 字距调整表 |
| `GPOS` | `gpos.go` | 字形定位表 |
| `GSUB` | `gsub.go` | 字形替换表 |

## 支持的字体格式

| 格式 | 文件 | 解析 | 序列化 | 说明 |
|------|------|------|--------|------|
| TTF | `ttf.go` / `serialize.go` | `Parse()` | `Serialize()` | TrueType 字体 |
| WOFF | `woff.go` | `ParseWOFF()` | `SerializeWOFF()` | zlib 逐表压缩 |
| WOFF2 | `woff2.go` | `ParseWOFF2()` | `SerializeWOFF2()` | Brotli 整体压缩 |
| EOT | `eot.go` | `ParseEOT()` | `SerializeEOT()` | 微软嵌入式字体 |

## 运行测试

```bash
# 运行全部测试
go test ./...

# 运行指定测试
go test -run TestRemoveGlyphs ./...

# 详细输出
go test -v ./...
```

## 许可证

MIT License
