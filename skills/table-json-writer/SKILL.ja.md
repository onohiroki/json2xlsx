---
name: table-json-writer
description: 高レベルな table JSON を出力し，`json2xlsx --table-json` で XLSX に変換するためのスキル．
---

# table-json-writer

単純な表・レポート・集計シートを求められたときに使います．出力は **table JSON のみ**にしてください．グラフ・画像・条件付き書式・スパークラインなどの高度な機能が必要な場合は `sheetjs-json-writer` を使ってください．

## 絶対ルール
- 表オブジェクトそのものを出力する．通常のチャットでは JSON のみを出力し，Markdown のコードフェンスで囲まない．MCP を使う場合は `json2xlsx_convert` の `table` 引数として渡す．
- トップレベルは `tables` キーにする．
- 単純なデータは `columns[].key` を使ったオブジェクト行で書く．
- 同じ数式を各行に繰り返さず，`columns[].formula` の `{key}` を使う．
- 生成後は `json2xlsx_validate` を呼び，問題をすべて直す．可能なら `json2xlsx_preview` で表示を確認する．

## 最小例
```json
{
  "tables": [
    {
      "name": "Sales",
      "columns": [
        { "key": "商品", "header": "商品", "width": 18 },
        { "key": "数量", "header": "数量", "format": "#,##0" },
        { "key": "単価", "header": "単価", "format": "#,##0" },
        { "key": "金額", "header": "金額", "format": "#,##0", "formula": "={数量}*{単価}" }
      ],
      "rows": [
        { "商品": "A", "数量": 10, "単価": 100 },
        { "商品": "B", "数量": 5,  "単価": 200 }
      ]
    }
  ]
}
```

## フィールド
| フィールド | 意味 |
|---|---|
| `tables` | 必須．1 要素が 1 シートに対応する． |
| `name` | シート名．省略時は `Sheet1`，`Sheet2` など． |
| `header` | 省略可．指定時は 1 行目がヘッダになり，データは 2 行目から始まる．`columns[].header` がある列はそちらが優先される． |
| `rows` | 必須．各行を配列，または `columns[].key` をキーにしたオブジェクトで表す． |
| `columns` | 省略可．index 0 が列 A． |
| `columns[].key` | 数式参照用の列キー．オブジェクト行では必須．日本語可． |
| `columns[].header` | この列のヘッダ文字列．トップレベルの `header` より優先される． |
| `columns[].width` | 列幅．`0 < width <= 255`． |
| `columns[].format` | `#,##0` や `0.00%` などの数値書式． |
| `columns[].align` | `left`，`center`，`right`． |
| `columns[].style` | `styles` の名前． |
| `columns[].formula` | データ行に適用する数式テンプレート．`{key}` は現在行のセル，`{key:col}` は列記号，`{r}` は行番号に置換される． |
| `styles` | 名前付きスタイル． |
| `headerStyle` / `bodyStyle` | ヘッダ行・データ行に適用するスタイル名． |
| `freezeHeader` | ヘッダ行を固定する．`header` が必要． |
| `autoFilter` | ヘッダとデータにオートフィルタを設定する．`header` が必要． |
| `merges` | `A1:D1` などの結合範囲． |

## セルの値
- 数値・文字列・真偽値・`null` はそのまま書き込まれる．
- `=` で始まる文字列は数式になる．
- 数式中の `{key}` は現在行のセル（`B2`），`{key:col}` は列記号（`B`），`{r}` は行番号（`2`）になる．
- 例: `={数量}*{単価}` は `=B2*C2`，`=SUM({金額:col}2:{金額:col}6)` は `=SUM(D2:D6)` になる．
- クロスシート参照も使える: `=SUM(Sales!D2:D5)` や `=SUM('売上'!D2:D5)`．空白や非 ASCII 文字を含むシート名はクォートする．
- 未知のプレースホルダは検証エラーになる．数式内にリテラルの波括弧が必要なら SheetJS 形式を使う．
- `=` で始まるリテラル文字列は `{ "v": "=text" }` を使う．
- 計算済みの値も持たせる場合は `{ "v": 42, "f": "SUM(A1:A3)" }` を使う．
- セルオブジェクトのフィールドは `v`，`f`，`format`，`style`，`align`，`t`．
- `v` にオブジェクトや配列を入れない．

## スタイル
```json
{
  "styles": {
    "header": {
      "font": { "bold": true, "color": "#000000" },
      "fill": { "type": "pattern", "pattern": 1, "color": ["#E0EBF5"] },
      "border": [{ "style": "thin", "color": "#000000" }],
      "alignment": { "horizontal": "center" }
    },
    "body": {
      "border": [{ "style": "thin", "color": "#000000" }]
    }
  },
  "headerStyle": "header",
  "bodyStyle": "body"
}
```
- 色は `#RRGGBB` 形式．
- 罫線スタイルは `thin`，`medium`，`thick`，`dashed`，`dotted`，`double`，`hair` など．
- `border[].side` を省略すると四辺すべてに適用される．
- スタイルの優先順位は `bodyStyle`/`headerStyle` < `columns[].style` < セルの `style`．
- `format` と `align` は選んだスタイルの上に適用される．

## よくある間違い
- table の `styles` に `id` を書かない．ID は自動で割り当てられる．
- セルオブジェクトの `f` に `=` を付けない．文字列値の先頭の `=` が数式の印になる．
- `freezeHeader` や `autoFilter` を使うときは `header` を用意する．
- 配列行とオブジェクト行を同じ表で混在させない．
- オブジェクト行ではすべての列に `key` が必要．キーに空白・`{`・`}`・`:` は使えない．
- シート名は 31 文字以内にし，`[]:*?/\` を含めない．
