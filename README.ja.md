# json2xlsx

[ [English](README.md) | 日本語 ]

`json2xlsx` は，SheetJS風の JSON または二次元配列形式の JSON から Excel の `.xlsx` ファイルを生成する Go 製 CLI ツールです．Go 側は **JSON → XLSX 変換だけ** を担当し，AI 呼び出しは含みません．[1][2]

あわせて，AI に SheetJS 風 JSON を安定して出力させるためのスキル `sheetjs-json-writer`（`skills/` ディレクトリに同梱）を併用する想定です．[3][4]

## 目的

このプロジェクトの目的は，次の 2 段階を分離して扱うことです．

1. AI が表や集計内容を **SheetJS 風 JSON** として出力する．
2. `json2xlsx` がその JSON を読み取り，`.xlsx` に変換する．[5][6]

全体を通して JSON は SheetJS 風を扱いますが，外部ツールとのデータやり取りの利便性のために，二次元配列形式の JSON の入力も受け付けます．

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
- SheetJS 風の Cell Object に加え，シンプルな二次元配列形式の JSON を入力できる．[4][3]
- 基本表，数式，改行，枠線，色，数値書式，リンク，固定ペインなどをサポート（二次元配列形式は数式やスタイルには非対応）．[6][9][10]
- AI 生成部を切り離しているため，任意の LLM と組み合わせやすい．[11][12]
- **グラフ生成** — 棒・折れ線・円・散布図など 8 種類のチャートをサポート．タイトル・凡例・軸・データラベルも設定可能．
- **日本語対応** — 系列名に日本語（例: `予算`，`実績`）が含まれていても Excel 凡例に正しく表示されます．

## インストール

### `go install`（推奨）

```bash
go install github.com/onohiroki/json2xlsx/cmd/json2xlsx@latest
```

### ローカルビルド

```bash
git clone git@github.com:onohiroki/json2xlsx.git
cd json2xlsx
go build -o json2xlsx ./cmd/json2xlsx
```

## 使い方

`json2xlsx` は単一バイナリの CLI で，XLSX と JSON を相互変換します．サブコマンドは `to-json` (XLSX → JSON)，`to-xlsx` (JSON → XLSX)，`to-md` (JSON / XLSX → Markdown テーブル)，`to-html` (JSON / XLSX → HTML `<table>`)，`to-csv` (JSON / XLSX → CSV) の 5 種類で，**サブコマンドを省略した場合は `to-xlsx` として動作**します．

### `to-json` — XLSX → JSON

XLSX を読み込み，`json2xlsx` に入力可能な JSON (セルマップ形式) を出力します．

```bash
json2xlsx to-json -i input.xlsx -o output.json
json2xlsx to-json -i input.xlsx -o output.json --date-display
json2xlsx to-json -i input.xlsx -o output.json --date-rfc3339
```

`-i` を省略すると標準入力，`-o` を省略すると標準出力を使います．

```bash
cat input.xlsx | json2xlsx to-json > output.json
```

日時セル (`t: "d"`) は，デフォルトでは Excel の内部シリアル値を数値として `v` に出力します．
`--date-display` を指定すると Excel の表示文字列を，`--date-rfc3339` を指定すると RFC3339 (UTC) 文字列を `v` に出力します．
`--date-serial` を明示的に指定してもデフォルトと同じ動作です（互換性のためのエイリアス）．
時刻だけの値 (`9:05`) は **日付なしの時刻** として扱います．

### `to-xlsx` — JSON → XLSX (デフォルト)

JSON を読み込み，`.xlsx` を出力します．デフォルトでは SheetJS 形式の Workbook JSON を受け付けます．`--data-json` を指定すると，二次元配列（例: `[["A", 1], ["B", 2]]`），オブジェクト配列，Map-of-Arrays の各形式に対応します．データ JSON 形式（`--data-json`）では数式やスタイルは指定できません．

```bash
json2xlsx to-xlsx -i input.json -o output.xlsx
json2xlsx to-xlsx -i data.json -o output.xlsx --data-json
```

`--compute` を指定すると，キャッシュ値 (`v`) がない数式 (`t: "f"`) を組み込みの数式エンジンで評価します．評価に失敗したセルはスキップされ，警告が stderr に出力されます．

```bash
json2xlsx to-xlsx -i input.json -o output.xlsx --compute
```

### 数式エンジン

組み込みの数式エンジンはセル単位で数式を評価します．以下に対応一覧を示します．

**算術演算子:** `+`, `-`, `*`, `/`

**比較演算子:** `<`, `>`, `=`, `<=`, `>=`, `<>` (等しくない)．比較結果は `1` (真) または `0` (偽) です．

**対応関数:**

| 関数 | 説明 |
|------|------|
| `SUM(n1, n2, ...)` | 数値の合計 |
| `AVERAGE(n1, n2, ...)` | 算術平均 |
| `COUNT(n1, n2, ...)` | 数値セルの個数 |
| `COUNTA(n1, n2, ...)` | 空でないセルの個数 |
| `MIN(n1, n2, ...)` | 最小値 |
| `MAX(n1, n2, ...)` | 最大値 |
| `ABS(x)` | 絶対値 |
| `ROUND(x, digits)` | 指定桁数に丸める |
| `ROUNDUP(x, digits)` | 切り上げ |
| `ROUNDDOWN(x, digits)` | 切り捨て |
| `INT(x)` | 整数部 |
| `TRUNC(x, digits)` | 0 に向かって切り詰め |
| `SIGN(x)` | x の符号 (-1, 0, または 1) |
| `PI()` | 円周率 (3.14159...) |
| `RAND()` | 0 以上 1 未満の乱数 |
| `PRODUCT(n1, n2, ...)` | 乗算 |
| `SUMPRODUCT(a1, a2, ...)` | 要素ごとの積の和 |
| `POWER(x, y)` | x の y 乗 |
| `SQRT(x)` | 平方根 |
| `LN(x)` | 自然対数 |
| `LOG(x, [base])` | 対数 (底省略で自然対数) |
| `LOG10(x)` | 常用対数 |
| `EXP(x)` | e の x 乗 |
| `MOD(x, y)` | x / y の剰余 |
| `FLOOR(x, significance)` | 指定した倍数に切り下げ |
| `CEILING(x, significance)` | 指定した倍数に切り上げ |
| `SIN(x)` | 正弦 (ラジアン) |
| `COS(x)` | 余弦 (ラジアン) |
| `TAN(x)` | 正接 (ラジアン) |
| `ASIN(x)` | 逆正弦 |
| `ACOS(x)` | 逆余弦 |
| `ATAN(x)` | 逆正接 |
| `ATAN2(x, y)` | y/x の逆正接 |
| `DEGREES(x)` | ラジアンを度に変換 |
| `RADIANS(x)` | 度をラジアンに変換 |
| `SINH(x)` | 双曲線正弦 |
| `COSH(x)` | 双曲線余弦 |
| `TANH(x)` | 双曲線正接 |
| `ASINH(x)` | 逆双曲線正弦 |
| `ACOSH(x)` | 逆双曲線余弦 |
| `ATANH(x)` | 逆双曲線正接 |
| `FACT(x)` | 階乗 |
| `MEDIAN(n1, n2, ...)` | 中央値 |
| `STDEV.S(n1, n2, ...)` | 標本標準偏差 |
| `STDEV.P(n1, n2, ...)` | 母標準偏差 |
| `VAR.S(n1, n2, ...)` | 標本分散 |
| `VAR.P(n1, n2, ...)` | 母分散 |
| `RANK(x, range, [order])` | 範囲内での順位 (0=降順, 非0=昇順) |
| `RANK.EQ(x, range, [order])` | RANK の別名 |
| `LARGE(range, k)` | k 番目に大きい値 |
| `SMALL(range, k)` | k 番目に小さい値 |
| `IF(cond, t_val, f_val)` | 条件分岐 (cond != 0 → t_val, それ以外 → f_val) |
| `IFERROR(expr, fallback)` | エラー時にフォールバック値を返す |
| `AND(n1, n2, ...)` | 論理積 (すべて非ゼロなら 1) |
| `OR(n1, n2, ...)` | 論理和 (いずれか非ゼロなら 1) |
| `NOT(x)` | 論理否定 (ゼロなら 1) |
| `SUMIF(check_range, criteria, [sum_range])` | 条件に合うセルの合計 |
| `COUNTIF(range, criteria)` | 条件に合うセルの個数 |
| `AVERAGEIF(check_range, criteria, [avg_range])` | 条件に合うセルの平均 |
| `SUMIFS(sum_range, crit_range1, crit1, ...)` | 複数条件での合計 |
| `COUNTIFS(crit_range1, crit1, ...)` | 複数条件での個数 |
| `AVERAGEIFS(avg_range, crit_range1, crit1, ...)` | 複数条件での平均 |
| `VLOOKUP(value, table, col_index)` | 縦方向検索 (完全一致) |
| `XLOOKUP(value, lookup_arr, return_arr, [not_found])` | 最新検索関数 (デフォルト値指定可) |
| `INDEX(range, row, [col])` | 指定行・列の値 |
| `MATCH(value, range, match_type)` | 範囲内の位置を返す |
| `CHOOSE(index, val1, val2, ...)` | インデックスで値を選択 |
| `TODAY()` | 現在日付をシリアル値で返す |
| `NOW()` | 現在日時をシリアル値で返す |

**制限:**

- 数値のみ対応しています．文字列関数 (`CONCAT`, `LEFT`, `FIND` など) や条件内の文字列比較は **利用できません**．
- 範囲参照 (`A1:A10` など) は関数の引数内でのみ有効です．単独の範囲はエラーになります．
- セル参照は A1 形式のみ対応 (R1C1 は不可)．列文字は 3 文字以内 (`A`–`ZZZ`)．
- シート間参照は **非対応** です．
- 配列数式，揮発性フラグ，反復計算は **非対応** です．
- 循環参照は検出されて警告として報告されます．その他の評価エラーは該当セルがスキップされます．

サブコマンド省略時も同じ動作になります．

```bash
json2xlsx -i input.json -o output.xlsx
```

標準入力からも受け付けます．

```bash
cat input.json | json2xlsx to-xlsx -o output.xlsx
```

### `to-md` — JSON / XLSX → Markdown

`Workbook` を Markdown のテーブルに変換します．入力は **JSON (json2xlsx 互換 Workbook) と XLSX の両方** に対応し，先頭 4 バイトの magic byte (`PK\x03\x04`) で自動判定します．AI への提示や `cat` での内容確認用の中間表現として使えます．`--compute` を指定すると，出力前に数式を評価できます．

```bash
json2xlsx to-md -i input.json -o output.md
json2xlsx to-md -i input.xlsx -o output.md
json2xlsx to-md -i input.json -o output.md --compute
cat input.xlsx | json2xlsx to-md > output.md
```

#### オプション

- `--mode f` (デフォルト): 数式があれば `=B1*2` を表示，無ければ `v` を表示．
- `--mode v`: `v` を優先表示．`v` が無ければ `=B1*2` にフォールバック．
- `--mode both`: `v` と数式の両方がある場合 `84<br />=B1*2` のように併記．
- `--first-row-header`: 最初の行をテーブルヘッダとして扱う．A/B/C 列名 + 行番号を抑制する．
- `--data-json`: JSON 入力をデータ JSON（二次元配列 / オブジェクト配列 / Map-of-Arrays）として扱う．

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
json2xlsx to-html -i input.json -o output.html --grid
cat input.xlsx | json2xlsx to-html > output.html
```

#### オプション

- `--mode v` (デフォルト): 数式セルは計算結果 (`v`) を表示．ヘッダ行なし．
- `--mode f`: 数式 (`=B2*C2`) を表示．A/B/C 列名 + 行番号を `<th>` で表示．
- `--mode both`: `v` と数式の両方を `v<br />=f` のように併記．
- `--data-json`: JSON 入力をデータ JSON（二次元配列 / オブジェクト配列 / Map-of-Arrays）として扱う．
- `--grid`: 空セルを含む全セルに灰色の細枠線を表示する（cellspacing を collapsed にする）．
- `--compute`: 出力前に数式を評価する．

#### スタイル反映

- 背景色 (Fill) → `background-color`
- 太字 (Font.Bold) → `font-weight: bold`
- 文字色 (Font.Color) → `color`
- 斜体 (Font.Italic) → `font-style: italic`
- フォントサイズ (Font.Size) → `font-size`
- 配置 (Alignment) → `text-align`, `vertical-align`, `white-space`
- 罫線 (Border) → `border`

### `to-csv` — JSON / XLSX -> CSV

JSON または XLSX を CSV に変換します．json2xlsx 形式の JSON，`csvtk csv2json` の出力，`xlsx-cli -j` の出力などをサポートしています．`--data-json` を指定すると，二次元配列・オブジェクト配列・Map-of-Arrays の各形式にも対応します．

```bash
json2xlsx to-csv -i input.json -o output.csv
json2xlsx to-csv -i data.json -o output.csv --data-json
json2xlsx to-csv -i input.xlsx -o output.csv --sheet "Sheet1"
json2xlsx to-csv -i input.xlsx -o output.csv --sheet-index 1
cat input.json | json2xlsx to-csv > output.csv
```

オプション:

- `--sheet`: シート名で特定のシートを抽出する（複数シートの XLSX / Workbook JSON 用）．
- `--sheet-index`: 1 始まりのインデックスでシートを抽出する（複数シートの XLSX / Workbook JSON 用）．
- `--compute`: 出力前に数式を評価する．

## 入力JSONの考え方

`json2xlsx` は，SheetJS 互換を意識した JSON を受け取り，セル単位で `.xlsx` に変換します．[3][4]

入力表現は 2 つのモードがあります．

**デフォルト（`--data-json` なし）:** SheetJS 形式の Workbook / Sheet JSON（セル参照形式または Cell Object 形式）を受け付けます．

**`--data-json` 指定時:** 以下のデータ指向形式を受け付けます．
- 二次元配列形式の JSON（例: `[["ヘッダ1", "ヘッダ2"], ["値1", 123]]`）
- オブジェクト配列形式（例: `[{"Name": "Alice", "Age": 30}, ...]`）
- Map-of-Arrays 形式（例: `{"Name": ["Alice", "Bob"], "Age": [30, 25]}`）

※ データ JSON 形式は純粋なデータ用であり，数式やスタイルはサポートされません．数式やスタイルを利用する場合は SheetJS 形式の Cell Object 形式を使用してください．

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

### 例: マップ・オブ・アレイ形式

キーが JSON 宣言順に保持されてヘッダ行になり，各配列が列データになります．配列の長さが異なる場合は `null` で埋められます．

```json
{
  "name": ["Alice", "Bob", "Carol"],
  "age":  [30,      25,    41],
  "city": ["Tokyo", "Osaka", "Nagoya"]
}
```

この入力は次のテーブルに変換されます．

| name | age | city |
|------|-----|------|
| Alice | 30 | Tokyo |
| Bob | 25 | Osaka |
| Carol | 41 | Nagoya |

詳しい例は `samples/table_map_of_array.json` を参照してください．

## グラフ

`book` ラッパー形式でのみグラフに対応しています．`charts` 配列にグラフオブジェクトを指定します．

### グラフオブジェクトのフィールド

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `id` | string | グラフ識別子 |
| `t` | string | `"chart"` 固定 |
| `mode` | string | `"embedded"`（デフォルト，シート内に埋め込み）または `"chartSheet"`（専用チャートシート） |
| `ct` | string | チャート種別: `"col"`, `"bar"`, `"line"`, `"area"`, `"pie"`, `"doughnut"`, `"scatter"`, `"radar"` |
| `sheet` | string | 埋め込み先のシート名 |
| `anchor` | string | アンカーセル（例: `"E2"`） |
| `dim` | object | `{w, h, offx, offy, sx, sy}` — 幅/高さ(px)，オフセット(EMU)，拡大率 |
| `title` | object | `{tx, overlay}` — グラフタイトル文字列とオーバーレイフラグ |
| `legend` | object | `{show, pos}` — `pos`: `"top"`, `"bottom"`, `"left"`, `"right"`, `"topRight"` |
| `xAxis` | object | `{title, minimum, maximum, majorUnit, minorUnit, reverseOrder, majorGridLines, minorGridLines, numFmt}` |
| `yAxis` | object | `xAxis` と同じ構造 |
| `plot` | object | `{varyColors, showBlanksAs}` — プロットエリアオプション |
| `ser` | array | 系列オブジェクトの配列（下記参照） |

### 系列オブジェクトのフィールド

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `name` | string | 系列名（リテラル文字列，または `"Sheet1!$A$1"` 形式のセル参照） |
| `cat` | string | カテゴリ範囲（例: `"部門予算!$A$2:$A$7"`） |
| `val` | string | 値範囲（例: `"部門予算!$B$2:$B$7"`） |
| `xVal` | string | X 値範囲（散布図のみ） |
| `yVal` | string | Y 値範囲（散布図のみ） |
| `bubble` | string | バブルサイズ範囲（バブルチャートのみ） |
| `line` | object | `{width}` — 線幅(pt) |
| `fill` | object | `{color}` — 塗りつぶし色（例: `"#FF0000"`） |
| `marker` | object | `{symbol, size}` — マーカー記号とサイズ |
| `dLbls` | object | `{showVal, showCatName, showSerName, showPercent, showLeaderLn}` — データラベル |

### 例

```json
{
  "version": "0.2",
  "book": {
    "sheets": {
      "部門予算": {
        "cells": {
          "A1": { "t": "s", "v": "部門" },
          "B1": { "t": "s", "v": "予算(百万円)" },
          "A2": { "t": "s", "v": "営業" }, "B2": { "t": "n", "v": 120 },
          "A3": { "t": "s", "v": "開発" }, "B3": { "t": "n", "v": 200 }
        }
      }
    },
    "charts": [
      {
        "id": "chart1",
        "t": "chart",
        "mode": "embedded",
        "ct": "col",
        "sheet": "部門予算",
        "anchor": "E2",
        "dim": { "w": 640, "h": 360 },
        "title": { "tx": "部門別予算と実績" },
        "legend": { "pos": "bottom" },
        "ser": [
          { "name": "予算", "cat": "部門予算!$A$2:$A$7", "val": "部門予算!$B$2:$B$7" },
          { "name": "実績", "cat": "部門予算!$A$2:$A$7", "val": "部門予算!$C$2:$C$7" }
        ]
      }
    ]
  }
}
```

詳細な例は `samples/chart_bar.json`，`samples/chart_scatter.json`，`samples/chart_timeseries.json` を参照してください．

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
| グラフ | Yes | 棒，折れ線，円，散布図など 8 種 |
| 固定ペイン | Yes | `freeze` フィールドで行/列の固定を指定 |
| リッチテキスト | No | 初期対象外 [4] |

## Goの依存パッケージ

現時点の依存関係は以下の通りです．

```go
require (
    github.com/xuri/excelize/v2 v2.8.1
    github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
    github.com/mattn/go-runewidth v0.0.24
    golang.org/x/text v0.14.0
)
```

`excelize` は Go で Excel ファイルを読み書きする代表的な OSS ライブラリ，`jsonschema` は JSON Schema バリデーション，`go-runewidth` は Markdown 出力時の文字幅計算，`golang.org/x/text` はエラーメッセージの多言語対応に使用しています．[2][1]

## Goのデータ構造

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

type ColInfo struct {
    Col   string  `json:"col"`
    Width float64 `json:"width"`
}

type RowInfo struct {
    Row    int     `json:"row"`
    Height float64 `json:"height"`
}

type FreezePane struct {
    Row int `json:"row,omitempty"`
    Col int `json:"col,omitempty"`
}
```

## `sheetjs-json-writer` との関係

`skills/` ディレクトリに同梱されている `sheetjs-json-writer` は，AI に対して次のような制約を与えるための SKILL.md です．

- JSON だけを出力する．
- Markdown の説明を付けない．
- `t`, `v`, `f`, `s` などのフィールドを正しく使う．
- 数式，改行，スタイルを決まった形式で出す．[4][6][3]

このため，`json2xlsx` 側は「正しい形式の JSON が来る」前提でシンプルに保てます．

また，SheetJS 互換の仕様を参考にした再実装は，互換実装として整理可能です．[5][4]

## ライセンス

このツールは **MIT ライセンス** のもとで公開されています。

主要な依存ライブラリのライセンスは以下の通りです：
- `excelize`: BSD 3-Clause
- `jsonschema/v6`: MIT
- `go-runewidth`: MIT

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
