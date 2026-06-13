# json2xlsx

[ [English](README.md) | 日本語 ]

`json2xlsx` は，SheetJS風の JSON から Excel の `.xlsx` ファイルを生成する Go 製 CLI ツールです．Go 側は **JSON → XLSX 変換だけ** を担当し，AI 呼び出しは含みません．[1][2]

あわせて，AI に SheetJS 風 JSON を安定して出力させるためのスキル `sheetjs-json-writer` を併用する想定です．[3][4]

## 目的

このプロジェクトの目的は，次の 2 段階を分離して扱うことです．

1. AI が表や集計内容を **SheetJS 風 JSON** として出力する．
2. `json2xlsx` がその JSON を読み取り，`.xlsx` に変換する．[5][6]

この分離により，Go ツールは軽量・テスト容易・OSS 公開しやすい構成になります．[7][8]

### なぜ `json2xlsx` を使うのか — AI 生成コードの非決定性を排除する

AI コーディングツールに「その場で XLSX を生成するコードを書かせる」ことでも一見同じ目的は達成できますが，以下の問題があります：

- **非決定性** — 同じプロンプトでも毎回異なるコード・異なる出力になり得る．品質の再現が保証されない．
- **実行リスク** — AI が生成したコードをその場で実行する必要がある．ライブラリのインストール・バージョン競合・潜在的なセキュリティ問題が発生し得る．
- **デバッグ困難** — 出力がおかしいとき，「コードのどこが間違っているか」を人間が追う必要がある．

`json2xlsx` はこの問題を **「AI に JSON だけを出させる」** ことで解決します：

- **JSON は決定論的** — 同じ JSON なら常に同じ XLSX が得られる．AI の「ブレ」は JSON 生成に閉じ込められ，変換パイプラインは安定．
- **実行不要** — AI は JSON を出力するだけでよく，コードを実行する必要がない（JSON の検証は軽い）．
- **デバッグが人間に優しい** — JSON は人間が読んで編集できる．間違いがあれば直接修正して再変換できる．
- **ツールは独立テスト可能** — `json2xlsx` 自体の品質は一度担保すれば使い回せる．毎回テストする必要がない．

要するに **「AI にコードを書かせる」ではなく「AI にデータを出させる」** という設計思想で，LLM の非決定性を吸収するレイヤーとして JSON を挟んでいる点が本質的な違いです．

## 特徴

- Go 1.22+ で動作する軽量 CLI．[2]
- 主要依存は `excelize` (XLSX 読み書き) と `jsonschema/v6` (JSON バリデーション)．[1]
- SheetJS 風の Cell Object を意識した JSON を入力できる．[4][3]
- 基本表，数式，改行，枠線，色，数値書式，リンクなどを段階的にサポートできる．[6][9][10]
- AI 生成部を切り離しているため，任意の LLM と組み合わせやすい．[11][12]

## インストール方針

初期段階では，ローカルでビルドして利用する前提です．

```bash
git clone git@github.com:onohiroki/json2xlsx.git
cd json2xlsx
go build -o json2xlsx ./cmd/json2xlsx
```

将来的には `go install` 対応を想定します．

```bash
go install json2xlsx/cmd/json2xlsx@latest
```

## 使い方

`json2xlsx` は単一バイナリの CLI で，XLSX と JSON を相互変換します．サブコマンドは `to-json` (XLSX → JSON)，`to-xlsx` (JSON → XLSX)，`to-md` (JSON / XLSX → Markdown テーブル)，`to-html` (JSON / XLSX → HTML `<table>`)，`to-csv` (csvtk csv2json の逆変換) の 5 種類で，**サブコマンドを省略した場合は `to-xlsx` として動作**します．

### `to-json` — XLSX → JSON

XLSX を読み込み，`json2xlsx` に入力可能な JSON (セルマップ形式) を出力します．

```bash
json2xlsx to-json -i input.xlsx -o output.json
json2xlsx to-json -i input.xlsx -o output.json --date-serial
json2xlsx to-json -i input.xlsx -o output.json --date-display
json2xlsx to-json -i input.xlsx -o output.json --date-rfc3339
```

`-i` を省略すると標準入力，`-o` を省略すると標準出力を使います．

```bash
cat input.xlsx | json2xlsx to-json > output.json
```

日時セル (`t: "d"`) は，デフォルトでは Excel の内部シリアル値を `v` に出力します．
`--date-display` を指定した場合のみ，Excel の表示文字列を `v` に出力します．
`--date-rfc3339` を指定した場合のみ，シリアル値から RFC3339 (UTC) に再解釈した値を `v` に出力します．
`--date-serial` を指定した場合は，Excel の内部シリアル値をそのまま数値として出力します．
時刻だけの値 (`9:05`) は **日付なしの時刻** として扱います．

### `to-xlsx` — JSON → XLSX (デフォルト)

JSON を読み込み，`.xlsx` を出力します．`--sheet` でシート名未指定時のデフォルトを指定できます．

```bash
json2xlsx to-xlsx -i input.json -o output.xlsx --sheet Sheet1
```

サブコマンド省略時も同じ動作になります．

```bash
json2xlsx -i input.json -o output.xlsx
```

標準入力からも受け付けます．

```bash
cat input.json | json2xlsx to-xlsx -o output.xlsx
```

### `to-md` — JSON / XLSX → Markdown

`Workbook` を Markdown のテーブルに変換します．入力は **JSON (json2xlsx 互換 Workbook) と XLSX の両方** に対応し，先頭 4 バイトの magic byte (`PK\x03\x04`) で自動判定します．AI への提示や `cat` での内容確認用の中間表現として使えます．

```bash
json2xlsx to-md -i input.json -o output.md
json2xlsx to-md -i input.xlsx -o output.md
cat input.xlsx | json2xlsx to-md > output.md
```

#### オプション

- `--mode f` (デフォルト): 数式があれば `=B1*2` を表示，無ければ `v` を表示．
- `--mode v`: `v` を優先表示．`v` が無ければ `=B1*2` にフォールバック．
- `--mode both`: `v` と数式の両方がある場合 `84<br />=B1*2` のように併記．
- `--first-row-header`: 最初の行をテーブルヘッダとして扱う．A/B/C 列名 + 行番号を抑制する．

ロングオプションは `--name` 形式で表記しています．短い `-i` / `-o` はそのまま `-` 1 文字で指定します．`-mode` のようにハイフン 1 つでも受け付けますが，ドキュメント上の表記は `--` に統一しています．

#### 出力例

`--mode f` (デフォルト):

```text
|   | A | B | C | D |
| --- | --- | --- | --- | --- |
| 1 | 製品 | 数量 | 単価 | 合計 |
| 2 | 商品A | 100 | 5000 | =B2*C2 |
```

`--first-row-header` (最初の行をヘッダとして扱う):

```text
| 製品 | 数量 | 単価 | 合計 |
| --- | --- | --- | --- |
| 商品A | 100 | 5000 | =B2*C2 |
```

#### 複数シート

複数シートの `Workbook` を渡すと，シートごとに `## <シート名>` 見出し付きのテーブルが連結されます．単一シート時は見出しは省略されます．

#### 注意点

- セル内の `|`, `\`, 改行は GFM のテーブルセルとして安全な形 (`\|`, `\\`, `<br />`) にエスケープされます．
- スタイル (色・罫線・フォント)，列幅，行高は Markdown には反映されません．
- マージセルは左上セルの値のみ出力され，それ以外は空セルになります．

### `to-html` — JSON / XLSX → HTML `<table>`

`Workbook` を HTML の `<table>` フラグメントに変換します．入力判定は `to-md` と同様に magic byte で自動判定します．

```bash
json2xlsx to-html -i input.json -o output.html
json2xlsx to-html -i input.xlsx -o output.html
cat input.xlsx | json2xlsx to-html > output.html
```

#### オプション

- `--mode v` (デフォルト): 数式セルは計算結果 (`v`) を表示．ヘッダ行なし．
- `--mode f`: 数式 (`=B2*C2`) を表示．A/B/C 列名 + 行番号を `<th>` で表示．
- `--mode both`: `v` と数式の両方を `v<br />=f` のように併記．

#### スタイル反映

- 背景色 (Fill) → `background-color`
- 太字 (Font.Bold) → `font-weight: bold`
- 文字色 (Font.Color) → `color`
- 斜体 (Font.Italic) → `font-style: italic`
- フォントサイズ (Font.Size) → `font-size`
- 配置 (Alignment) → `text-align`, `vertical-align`, `white-space`
- 罫線 (Border) → `border`

### `to-csv` — csvtk / xlsx-cli JSON -> CSV

`csvtk csv2json` が出力する JSON (オブジェクト配列) と，`xlsx-cli -j` が出力する JSON を CSV に変換します．
`xlsx-cli -j` 形式は先頭シートのみ処理し，シート名行は無視します．シート名行の後に配列が無い場合はエラー終了します．
`to-json` が出力する json2xlsx 形式の JSON (Workbook オブジェクト) は受け付けず，エラー終了します．

```bash
json2xlsx to-csv -i input.json -o output.csv
cat input.json | json2xlsx to-csv > output.csv
```

## 入力JSONの考え方

`json2xlsx` は，SheetJS 互換を意識した JSON を受け取り，セル単位で `.xlsx` に変換します．[3][4]

想定する入力表現は次の 3 系統です．

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
| ハイパーリンク | Yes | `L` フィールドで指定 |
| マージセル | Yes | `merges` 配列で指定 |
| リッチテキスト | No | 初期対象外 [4] |

## Goの依存パッケージ

現時点の依存関係は以下の通りです．

```go
require (
    github.com/xuri/excelize/v2 v2.8.1
    github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
)
```

`excelize` は Go で Excel ファイルを読み書きする代表的な OSS ライブラリ，`jsonschema` は JSON Schema バリデーションに使用しています．[2][1]

## Goのデータ構造案

```go
type Cell struct {
    T string      `json:"t,omitempty"`
    V interface{} `json:"v,omitempty"`
    F string      `json:"f,omitempty"`
    Z string      `json:"z,omitempty"`
    S int         `json:"s,omitempty"`
    L interface{} `json:"l,omitempty"`
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

type Font struct {
    Name   string  `json:"name,omitempty"`
    Size   float64 `json:"size,omitempty"`
    Bold   bool    `json:"bold,omitempty"`
    Italic bool    `json:"italic,omitempty"`
    Color  string  `json:"color,omitempty"`
}

type Alignment struct {
    Horizontal string `json:"horizontal,omitempty"`
    Vertical   string `json:"vertical,omitempty"`
    WrapText   bool   `json:"wrapText,omitempty"`
}

type Style struct {
    ID        int        `json:"id"`
    Fill      *Fill      `json:"fill,omitempty"`
    Border    []Border   `json:"border,omitempty"`
    Font      *Font      `json:"font,omitempty"`
    Alignment *Alignment `json:"alignment,omitempty"`
    NumFmt    string     `json:"numFmt,omitempty"`
}
```

## `sheetjs-json-writer` との関係

別途用意する `sheetjs-json-writer` は，AI に対して次のような制約を与えるための SKILL.md です．

- JSON だけを出力する．
- Markdown の説明を付けない．
- `t`, `v`, `f`, `s` などのフィールドを正しく使う．
- 数式，改行，スタイルを決まった形式で出す．[4][6][3]

このため，`json2xlsx` 側は「正しい形式の JSON が来る」前提でシンプルに保てます．

## ライセンス方針

この構成は OSS として公開可能です．`excelize` は BSD 3-Clause，`jsonschema/v6` は Apache 2.0 ですが，本ツール自体は AI 呼び出しを含みません．

また，SheetJS 互換の仕様を参考にした再実装は，互換実装として整理可能です．[5][4]

## 開発状況

実装は次の順で進め，全項目完了しています．

1. ✅ JSON 読み込み
2. ✅ 基本表の出力
3. ✅ Cell Object 対応
4. ✅ 数式対応
5. ✅ スタイル対応
6. ✅ 改行・列幅・行高・リンク対応
7. ✅ テスト整備

## 今後の成果物

このリポジトリでは，以下の成果物を揃えています．

- ✅ `README.md`
- ✅ `SKILL.md` (`sheetjs-json-writer` 用)
- ✅ Go 実装本体
- ✅ サンプル JSON (test_data/ 以下)

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
