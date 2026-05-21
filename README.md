# sheet2xlsx

`sheet2xlsx` は、SheetJS風の JSON から Excel の `.xlsx` ファイルを生成する Go 製 CLI ツールです。Go 側は **JSON → XLSX 変換だけ** を担当し、AI 呼び出しは含みません。[1][2]

あわせて、AI に SheetJS 風 JSON を安定して出力させるためのスキル `sheetjs-json-writer` を併用する想定です。[3][4]

## 目的

このプロジェクトの目的は、次の 2 段階を分離して扱うことです。

1. AI が表や集計内容を **SheetJS 風 JSON** として出力する。
2. `sheet2xlsx` がその JSON を読み取り、`.xlsx` に変換する。[5][6]

この分離により、Go ツールは軽量・テスト容易・OSS 公開しやすい構成になります。[7][8]

## 特徴

- Go 1.22+ で動作する軽量 CLI。[2]
- 主要依存は `excelize` のみ。[1]
- SheetJS 風の Cell Object を意識した JSON を入力できる。[4][3]
- 基本表、数式、改行、枠線、色、数値書式、リンクなどを段階的にサポートできる。[6][9][10]
- AI 生成部を切り離しているため、任意の LLM と組み合わせやすい。[11][12]

## インストール方針

初期段階では、ローカルでビルドして利用する前提です。

```bash
git clone <your-repo-url>
cd sheet2xlsx
go build -o sheet2xlsx ./cmd/sheet2xlsx
```

将来的には `go install` 対応を想定します。

```bash
go install github.com/yourname/sheet2xlsx/cmd/sheet2xlsx@latest
```

## 使い方

`sheet2xlsx` は単一バイナリの CLI で、XLSX と JSON を相互変換します。サブコマンドは `to-json` (XLSX → JSON) と `to-xlsx` (JSON → XLSX) の 2 種類で、**サブコマンドを省略した場合は `to-json` として動作**します。

### `to-json` — XLSX → JSON (デフォルト)

XLSX を読み込み、`sheet2xlsx` に入力可能な JSON (セルマップ形式) を出力します。

```bash
sheet2xlsx to-json -i input.xlsx -o output.json
```

サブコマンド省略時も同じ動作になります。

```bash
sheet2xlsx -i input.xlsx -o output.json
```

`-i` を省略すると標準入力、`-o` を省略すると標準出力を使います。

```bash
cat input.xlsx | sheet2xlsx to-json > output.json
```

### `to-xlsx` — JSON → XLSX

JSON を読み込み、`.xlsx` を出力します。`-sheet` でシート名未指定時のデフォルトを指定できます。

```bash
sheet2xlsx to-xlsx -i input.json -o output.xlsx -sheet Sheet1
```

標準入力からも受け付けます。

```bash
cat input.json | sheet2xlsx to-xlsx -o output.xlsx
```

### `to-md` — JSON / XLSX → Markdown

`Workbook` を Markdown のテーブルに変換します。入力は **JSON (sheet2xlsx 互換 Workbook) と XLSX の両方** に対応し、先頭 4 バイトの magic byte (`PK\x03\x04`) で自動判定します。AI への提示や `cat` での内容確認用の中間表現として使えます。

```bash
sheet2xlsx to-md -i input.json -o output.md
sheet2xlsx to-md -i input.xlsx -o output.md
cat input.xlsx | sheet2xlsx to-md > output.md
```

#### オプション

- `-mode f` (デフォルト): 数式があれば `=B1*2` を表示、無ければ `v` を表示。
- `-mode v`: `v` を優先表示。`v` が無ければ `=B1*2` にフォールバック。
- `-mode both`: `v` と数式の両方がある場合 `84<br />=B1*2` のように併記。
- `-row-index`: 先頭に行番号列 (`1`, `2`, …) を追加。Excel と座標を照合したい時に便利。

#### 出力例 (`-mode f`)

```text
| A | B | C |
| --- | --- | --- |
| 100 | 5000 | =B2*C2 |
```

#### 複数シート

複数シートの `Workbook` を渡すと、シートごとに `## <シート名>` 見出し付きのテーブルが連結されます。単一シート時は見出しは省略されます。

#### 注意点

- セル内の `|`, `\`, 改行は GFM のテーブルセルとして安全な形 (`\|`, `\\`, `<br />`) にエスケープされます。
- スタイル (色・罫線・フォント)、列幅、行高は Markdown には反映されません。
- マージセルは左上セルの値のみ出力され、それ以外は空セルになります。

## 入力JSONの考え方

`sheet2xlsx` は、SheetJS 互換を意識した JSON を受け取り、セル単位で `.xlsx` に変換します。[3][4]

想定する入力表現は次の 3 系統です。

- 配列オブジェクト形式
- セル参照形式 (`A1`, `B2` など)
- Cell Object 形式

### 例: Cell Object 形式

```json
{
  "cells": {
    "A1": {"t": "s", "v": "製品"},
    "B1": {"t": "s", "v": "数量"},
    "C1": {"t": "s", "v": "単価"},
    "D1": {"t": "s", "v": "合計", "s": 1},
    "A2": {"t": "s", "v": "商品A\n特価"},
    "B2": {"t": "n", "v": 100},
    "C2": {"t": "n", "v": 5000},
    "D2": {"t": "f", "f": "B2*C2", "v": 500000, "s": 1}
  },
  "styles": [
    {
      "id": 1,
      "fill": {"type": "pattern", "pattern": 1, "color": ["#E0EBF5"]},
      "border": [{"style": "thin", "color": "#000000"}],
      "numFmt": "#,##0"
    }
  ]
}
```

## 対応予定機能

| 機能 | 初期対応 | 備考 |
|------|----------|------|
| 基本表生成 | Yes | 文字列・数値の配置 [1] |
| 数式 | Yes | セル参照式を想定 [6] |
| セル内改行 | Yes | `\n` を改行として扱う [10][13] |
| 枠線 | Yes | thin / medium など [9] |
| 背景色 | Yes | Fill を利用 [14] |
| 数値書式 | Yes | `z` / `numFmt` 相当 [15] |
| ハイパーリンク | Later | 段階対応 |
| マージセル | Later | 段階対応 |
| リッチテキスト | No | 初期対象外 [4] |

## Goの依存パッケージ

現時点の基本方針では、依存は最小限にします。

```go
require github.com/xuri/excelize/v2 v2.8.1
```

`excelize` は Go で Excel ファイルを読み書きする代表的な OSS ライブラリです。[2][1]

## Goのデータ構造案

```go
type Cell struct {
    T string      `json:"t"`
    V interface{} `json:"v"`
    F string      `json:"f,omitempty"`
    Z string      `json:"z,omitempty"`
    S int         `json:"s,omitempty"`
    L interface{} `json:"l,omitempty"`
    H float64     `json:"h,omitempty"`
    W float64     `json:"w,omitempty"`
}

type Fill struct {
    Type    string   `json:"type"`
    Pattern int      `json:"pattern"`
    Color   []string `json:"color"`
}

type Border struct {
    Style string `json:"style"`
    Color string `json:"color"`
    Side  string `json:"side,omitempty"`
}

type Style struct {
    ID     int      `json:"id"`
    Fill   *Fill    `json:"fill,omitempty"`
    Border []Border `json:"border,omitempty"`
    NumFmt string   `json:"numFmt,omitempty"`
}
```

## `sheetjs-json-writer` との関係

別途用意する `sheetjs-json-writer` は、AI に対して次のような制約を与えるための SKILL.md です。

- JSON だけを出力する。
- Markdown の説明を付けない。
- `t`, `v`, `f`, `s` などのフィールドを正しく使う。
- 数式、改行、スタイルを決まった形式で出す。[4][6][3]

このため、`sheet2xlsx` 側は「正しい形式の JSON が来る」前提でシンプルに保てます。

## ライセンス方針

この構成は OSS として公開可能です。`excelize` は BSD 3-Clause、`openai-go` は Apache 2.0 ですが、本ツール自体は AI 呼び出しを含まないため、主要依存は `excelize` のみです。[8][7]

また、SheetJS 互換の仕様を参考にした再実装は、互換実装として整理可能です。[5][4]

## 開発方針

実装は次の順で進める想定です。

1. JSON 読み込み
2. 基本表の出力
3. Cell Object 対応
4. 数式対応
5. スタイル対応
6. 改行・列幅・行高・リンク対応
7. テスト整備

## 今後の成果物

このリポジトリでは、将来的に以下を揃える想定です。

- `README.md`
- `SKILL.md` (`sheetjs-json-writer` 用)
- Go 実装本体
- サンプル JSON
- テストデータ
- GitHub Copilot Agent 用の計画書

## 参考

- Excelize documentation / package reference[1]
- Excelize repository[2]
- SheetJS API reference[3]
- SheetJS Cell Objects[4]
- SheetJS license[5]

情報源
[1] excelize package - github.com/xuri/excelize/v2 https://pkg.go.dev/github.com/xuri/excelize/v2
[2] qax-os/excelize: Go language library for reading and ... https://github.com/qax-os/excelize
[3] API Reference https://docs.sheetjs.com/docs/api/
[4] Cell Objects https://docs.sheetjs.com/docs/csf/cell/
[5] License https://docs.sheetjs.com/docs/miscellany/license/
[6] How do I save data using `json_to_sheet` to save formulas? https://github.com/SheetJS/sheetjs/issues/2017
[7] はじめに · Excelize ドキュメンテーション - Ri Xu Online https://xuri.me/excelize/ja/
[8] openai-go/LICENSE at main https://github.com/openai/openai-go/blob/main/LICENSE
[9] Cell · Excelize Document - Ri Xu Online https://xuri.me/excelize/en/cell.html
[10] セル · Excelize ドキュメンテーション - Ri Xu Online https://xuri.me/excelize/ja/cell.html
[11] How to write better prompts for GitHub Copilot https://github.blog/developer-skills/github/how-to-write-better-prompts-for-github-copilot/
[12] Best practices for using GitHub Copilot https://docs.github.com/en/copilot/get-started/best-practices
[13] How can i wraptext to the new row in excelize https://stackoverflow.com/questions/74632921/how-can-i-wraptext-to-the-new-row-in-excelize
[14] Style · Excelize Document - Ri Xu Online https://xuri.me/excelize/en/style.html
[15] Number Formats https://docs.sheetjs.com/docs/csf/features/nf/
