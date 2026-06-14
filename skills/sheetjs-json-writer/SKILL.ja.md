---
name: sheetjs-json-writer
description: AI に SheetJS 風 JSON のみを出力させ，`json2xlsx` で XLSX に変換できる形式に整える．
---

# sheetjs-json-writer

## 目的
ユーザーの自然文の表指示を受け取り，`json2xlsx` が解釈できる **SheetJS 風 JSON のみ** を出力する．

## 絶対ルール
- 出力は JSON だけ．説明文・前置き・後置きを書かない．
- Markdown のコードフェンス（トリプルバッククォート）を付けない．先頭文字は必ず `{`．
- JSON 以外の文字を 1 文字も含めない．
- **集計・分析・計算（平均・合計・サマリなど）を指示された場合は，事前計算値を並べるよりも Excel 数式を優先して記述する．セル参照を使った数式オブジェクト (`t:"f"`, `f:"SUM(A1:A3)"` など) を用い，XLSX 変換後にシート上で再計算可能にする．計算結果を `v` に併記しても良いが，数式 (`f`) の記述は必須とする．**

## トップレベル構造

3つの形式がある:

**フラット（単一シート）:**
```json
{"name":"Sheet1","cells":{...},"rows":[...],"cols":[...],"rowDims":[...],"merges":[...],"styles":[...]}
```

**フラット（複数シート）:**
```json
{"sheets":[{"name":"...","cells":{...}},...],"styles":[...]}
```

**book ラッパー（チャート対応）:**
```json
{
  "version": "0.2",
  "book": {
    "props": {},
    "sheets": {"Sheet1":{"cells":{...}}, "Sheet2":{"cells":{...}}},
    "charts": [],
    "styles": []
  }
}
```

`book.sheets` はキーがシート名のオブジェクト。`book.charts` にグラフ定義を記述できる。

## セルオブジェクト (cells)
キーは `A1` のようなセル参照．値は次のフィールドを持つ:

| キー | 意味 | 例 |
|------|------|-----|
| `t`  | セル型 `s` 文字列 / `n` 数値 / `b` 真偽 / `f` 数式 / `d` 日付 | `"n"` |
| `v`  | 値 | `100` |
| `f`  | 数式 (`t="f"` のとき) | `"B2*C2"` |
| `z`  | 数値書式コード | `"#,##0"` |
| `s`  | `styles[].id` への参照 | `1` |
| `l`  | ハイパーリンク (文字列 or `{target, tooltip}`) | `"https://..."` |

## 記法ルール
- 数式: `=` 不要．改行: `\n`（表示には `wrapText: true` が必要）．色: `#RRGGBB` のみ．スタイル参照: `s` フィールド．

## スタイル定義 (styles)
```json
{
  "id": 1,
  "fill":   {"type": "pattern", "pattern": 1, "color": ["#E0EBF5"]},
  "border": [{"style": "thin", "color": "#000000"}],
  "font":   {"name": "Calibri", "size": 11, "bold": true, "color": "#000000"},
  "alignment": {"horizontal": "center", "vertical": "center", "wrapText": true},
  "numFmt": "#,##0"
}
```
- `border[].side` を省略すると四辺すべてに適用する．`"left"|"right"|"top"|"bottom"` で個別指定可．
- 罫線スタイル: `thin`, `medium`, `thick`, `dashed`, `dotted`, `double` など．

## その他
- 列幅: `"cols": [{"col": "A", "width": 18}]`
- 行高: `"rowDims": [{"row": 1, "height": 24}]`
- マージ: `"merges": [{"range": "A1:B1"}]`

## 代表例: 売上表 (単一シート)
```json
{
  "name": "売上",
  "cells": {
    "A1":{"t":"s","v":"製品","s":2},"B1":{"t":"s","v":"数量","s":2},
    "C1":{"t":"s","v":"単価","s":2},"D1":{"t":"s","v":"合計","s":2},
    "A2":{"t":"s","v":"商品A\n特価"},"B2":{"t":"n","v":100},
    "C2":{"t":"n","v":5000,"s":1},"D2":{"t":"f","f":"B2*C2","s":1}
  },
  "styles":[
    {"id":1,"numFmt":"#,##0","border":[{"style":"thin","color":"#000000"}]},
    {"id":2,"fill":{"type":"pattern","pattern":1,"color":["#E0EBF5"]},"border":[{"style":"thin","color":"#000000"}],"font":{"bold":true},"alignment":{"horizontal":"center","wrapText":true}}
  ]
}
```

## 代表例: 売上管理 (複数シート)
```json
{
  "sheets": [
    {
      "name": "売上",
      "cells": {
        "A1":{"t":"s","v":"製品","s":2},"B1":{"t":"s","v":"数量","s":2},
        "C1":{"t":"s","v":"単価","s":2},"D1":{"t":"s","v":"合計","s":2},
        "A2":{"t":"s","v":"商品A"},"B2":{"t":"n","v":100},
        "C2":{"t":"n","v":5000,"s":1},"D2":{"t":"f","f":"B2*C2","s":1}
      },
      "merges":[],
      "rowDims":[],
      "cols":[{"col":"A","width":14},{"col":"B","width":10},{"col":"C","width":10},{"col":"D","width":12}]
    },
    {
      "name": "商品マスタ",
      "cells": {
        "A1":{"t":"s","v":"商品コード","s":2},"B1":{"t":"s","v":"商品名","s":2},
        "C1":{"t":"s","v":"単価","s":2},
        "A2":{"t":"s","v":"A-001"},"B2":{"t":"s","v":"商品A"},
        "C2":{"t":"n","v":5000,"s":1}
      }
    }
  ],
  "styles":[
    {"id":1,"numFmt":"#,##0","border":[{"style":"thin","color":"#000000"}]},
    {"id":2,"fill":{"type":"pattern","pattern":1,"color":["#E0EBF5"]},"border":[{"style":"thin","color":"#000000"}],"font":{"bold":true},"alignment":{"horizontal":"center","wrapText":true}}
  ]
}
```

## グラフ (charts)

book ラッパー形式の `book.charts` にグラフオブジェクトの配列を記述できる。

| フィールド | 型 | 必須 | 説明 |
|-----------|----|------|------|
| `id`    | string | 任意 | グラフ識別 ID |
| `t`     | string | 任意 | グラフの場合は `"chart"` |
| `mode`  | string | 任意 | `"embedded"`(シート埋め込み) または `"chartSheet"`(専用シート) |
| `ct`    | string | 必須 | `col`, `bar`, `line`, `area`, `pie`, `doughnut`, `scatter`, `radar`, `combo` |
| `sheet` | string | 必須 | 配置先シート名 |
| `anchor`| string | embedded のみ | 左上セルアドレス (A1) |
| `ser`   | array  | 必須 | データ系列の配列（1 つ以上） |
| `dim`   | object | 任意 | 幅・高さ (`w`, `h`, `offx`, `offy`) |
| `title` | object | 任意 | `{ "tx": "タイトル" }` |
| `legend`| object | 任意 | `{ "show": true, "pos": "bottom" }` |
| `plot`  | object | 任意 | プロット領域オプション |
| `xAxis` | object | 任意 | X 軸設定 |
| `yAxis` | object | 任意 | Y 軸設定 |

系列 (`ser[]`) のフィールド:

| フィールド | 型 | 説明 |
|-----------|----|------|
| `name` | string | 系列名 |
| `cat`  | string | カテゴリ範囲 (A1, 例: `"Sheet1!$A$2:$A$13"`) |
| `val`  | string | 値範囲 (A1) |
| `line` | object | 線スタイル (`{ "width": 1.5 }`) |
| `fill` | object | 塗りスタイル (`{ "color": "#4472C4" }`) |

例（埋め込み）:
```json
{
  "version": "0.2",
  "book": {
    "sheets": {
      "Sheet1": { "cells": { "A1": {"t":"s","v":"月"}, "B1": {"t":"s","v":"売上"} } }
    },
    "charts": [
      {
        "id": "ch1",
        "t": "chart",
        "mode": "embedded",
        "ct": "col",
        "sheet": "Sheet1",
        "anchor": "D2",
        "title": { "tx": "月次売上" },
        "ser": [
          { "name": "売上", "cat": "Sheet1!$A$2:$A$13", "val": "Sheet1!$B$2:$B$13" }
        ]
      }
    ]
  }
}
```

chartSheet モードでは `mode` を `"chartSheet"` にし，`anchor` を省略し，新しいシート名を `sheet` に指定する．

## json2xlsx による検証
- 出力前に検証する:
  ```
  json2xlsx -i output.json > /dev/null      # Linux / macOS
  json2xlsx -i output.json                  # Windows PowerShell
  ```
- エラーがあれば修正し，終了コード 0 になるまで再テストする．
- よくあるエラー: **無効なセル名**（A-Z, AA-AZ を使用），**許可されていないプロパティ**（トップレベル構造が誤っている），**必須プロパティ欠落**（`cells` キーが必要），**無効な enum 値**（例: `"bottom"` が正しく `"b"` は不可）．

## エラー時の自己修復
- もし JSON 構文エラー，不明スキーマフィールド，未定義スタイル ID を検出した場合は，説明文を出さずに **修正版 JSON だけを再出力** する．
