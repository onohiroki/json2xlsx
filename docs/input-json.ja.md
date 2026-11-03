# 入力JSONの比較

[ [English](input-json.md) | 日本語 ]

`json2xlsx` は，目的の異なる 3 系統の入力 JSON を受け付けます．このドキュメントでは，それぞれの違いと使い分けを実装の挙動に沿って説明します．

- [README に戻る](../README.ja.md)
- [English version](input-json.md)

## 1. 3系統の概要

| 系統 | 指定方法 | 入力の形 | 主な用途 |
|---|---|---|---|
| **SheetJS 形式** | フラグなし | `cells` のセルマップ，`sheets` 配列，`book` ラッパー | 凝った帳票，グラフ，画像，XLSX 往復 |
| **データ JSON** | `--data-json` | 二次元配列，オブジェクト配列，Map-of-Arrays | CSV / SQL / API 由来の生データ |
| **table JSON** | `--table-json` | `tables` / `header` / `rows` / `columns` | 一般的な表，レポート，集計シート |

`--data-json` と `--table-json` は排他です．同時に指定するとエラーになります．

## 2. どれを選ぶか

1. **グラフ・画像・条件付き書式・スパークライン・セル単位の細かい配置が必要**  
   → **SheetJS 形式**を使います．XLSX を `to-json` した結果をそのまま再変換する場合もこちらです．

2. **ヘッダ，列幅，数値書式，数式テンプレートを使って普通の表を作りたい**  
   → **table JSON**を使います．LLM に生成させる場合も，セル座標を数える必要がないためこちらが向いています．

3. **値だけを渡して，装飾や数式は不要**  
   → **データ JSON**を使います．CSV，SQL の結果，API レスポンスをそのまま渡す用途です．

判断に迷う場合は，次の順で考えます．

```
グラフ・画像・条件付き書式・スパークラインが必要か？
  └─ はい → SheetJS 形式
       └─ いいえ → ヘッダ・列書式・数式テンプレートが必要か？
            └─ はい → table JSON
                 └─ いいえ → データ JSON
```

## 3. 同じ表を3形式で書く

次の表を例にします．

| 商品 | 数量 | 単価 | 金額 |
|---|---:|---:|---:|
| A | 10 | 100 | 数式 |
| B | 5 | 200 | 数式 |

### SheetJS 形式

```json
{
  "cells": {
    "A1": {"t": "s", "v": "商品"},
    "B1": {"t": "s", "v": "数量"},
    "C1": {"t": "s", "v": "単価"},
    "D1": {"t": "s", "v": "金額"},
    "A2": {"t": "s", "v": "A"},
    "B2": {"t": "n", "v": 10},
    "C2": {"t": "n", "v": 100},
    "D2": {"t": "f", "f": "B2*C2"},
    "A3": {"t": "s", "v": "B"},
    "B3": {"t": "n", "v": 5},
    "C3": {"t": "n", "v": 200},
    "D3": {"t": "f", "f": "B3*C3"}
  }
}
```

### table JSON

```json
{
  "tables": [
    {
      "columns": [
        {"key": "商品", "header": "商品"},
        {"key": "数量", "header": "数量"},
        {"key": "単価", "header": "単価"},
        {"key": "金額", "header": "金額", "formula": "={数量}*{単価}"}
      ],
      "rows": [
        {"商品": "A", "数量": 10, "単価": 100},
        {"商品": "B", "数量": 5, "単価": 200}
      ]
    }
  ]
}
```

### データ JSON（二次元配列）

```json
[["商品", "数量", "単価"], ["A", 10, 100], ["B", 5, 200]]
```

データ JSON は値専用のため，金額列のような数式列は表現できません．数式や書式が必要な場合は table JSON か SheetJS 形式を使ってください．

### 出力の比較

SheetJS 形式と table JSON は，`to-md --first-row-header` で同じ結果になります．

```markdown
| 商品 | 数量 | 単価 | 金額   |
| ---- | ---: | ---: | ------ |
| A    |   10 |  100 | =B2*C2 |
| B    |    5 |  200 | =B3*C3 |
```

## 4. 変換パイプライン

3 系統は，入力の解釈だけが異なり，最終的には同じ `Workbook` に合流します．

```
JSON + フラグ
├─ SheetJS → UnmarshalWorkbook(false)
│             ├─ SheetNames / Sheets 形式の正規化
│             ├─ !ref などの内部キー除去
│             └─ json.Unmarshal
├─ データ JSON → UnmarshalWorkbook(true)
│             └─ 二次元配列 / オブジェクト配列 / Map-of-Arrays を自動判別
└─ table JSON → UnmarshalTableJSON
              └─ フィールド検証と意味検証

                ↓ normalizeWorkbook
        Workbook{Sheets, Styles}
                ↓
   writeSheet / BuildCellGrid → excelize
```

コマンドごとの入口は次のとおりです．

| コマンド | 入口 | 備考 |
|---|---|---|
| `to-xlsx` | `ParseWorkbook` | 3 系統すべて対応 |
| `to-md` / `to-html` | `ReadWorkbookWithFormat` | 3 系統すべて対応 |
| `to-csv` | `ToCSVWithFormat` | SheetJS は `UnmarshalWorkbook` に統一済み |
| `info` | `ReadBookInfoSourceWithFormat` | JSON 形式を自動判定 |
| MCP | `loadWorkbook` | `input_path` は `DetectJSONFormat` で自動判定 |

`to-csv` は，フラグなしの配列を従来の `xlsx-cli` / `csvtk` 互換経路で処理します．データ JSON として扱う場合は `--data-json` を明示してください．

## 5. コマンド別対応表

`✓` は対応，`✗` は非対応，`△` は条件付きです．

| 機能 | `to-xlsx` | `to-md` | `to-html` | `to-csv` | `info` | MCP |
|---|---|---|---|---|---|---|
| SheetJS 形式 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| データ JSON | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| table JSON | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| JSON 形式の自動判定 | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ |
| `--range` | ✗ | ✓ | ✓ | ✓ | ✗ | preview のみ |
| シート指定 | ✗ | ✗ | ✗ | ✓ | `--sheet` | inspect のみ |
| `--compute` | ✓ | ✓ | ✓ | ✓ | ✗ | ✓ |
| `--autofit` | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |

補足:

- `to-xlsx` はブック全体を変換するため，範囲やシートの指定はありません．
- `to-md` / `to-html` は複数シートをすべて出力します．シート見出しは `to-md` のみ表示されます．
- `to-csv` は `--sheet` / `--sheet-index` で 1 シートを選びます．データ JSON では実効名 `Sheet1` も指定できます．
- `info` は JSON の形式を自動判定します．`--data-json` / `--table-json` で上書きできます．
- MCP の `json2xlsx_preview` は `range`，`json2xlsx_inspect` は `sheet` を受け付けます．

## 6. 挙動の違い

### 6.1 シート順とシート名

- `book` ラッパーの `book.sheets` はオブジェクトです．順序を保証するには `book.sheetOrder` を指定します．
- `book.sheetOrder` がない場合，シートはシート名のアルファベット順になります．
- `to-json` は XLSX のシート順を `book.sheetOrder` として出力します．
- 重複するシート名は，大文字小文字を無視してエラーになります．
- シート名が空の場合，実効名は `Sheet1`，`Sheet2` などになります．`info` や `to-md` の見出しもこの実効名を使います．

### 6.2 ヘッダと列

| 系統 | ヘッダ | 列数 | 欠落の扱い |
|---|---|---|---|
| SheetJS | 概念なし（A1 などに自分で置く） | `cells` の最大座標 / `rows` の最大長 | 空セル |
| データ JSON（二次元配列） | なし（1 行目もデータ） | 最大行長 | 空セル |
| データ JSON（オブジェクト配列） | キーの初出順で自動生成 | 全キーの和集合 | `null` |
| データ JSON（Map-of-Arrays） | キーの宣言順 | キー数 | `null` |
| table JSON | `header` または `columns[].header` | `header` / `columns` / 最長行の最大 | 空セル |

補足:

- データ JSON のオブジェクト配列は，入力でのキーの出現順が列順になります．
- table JSON のオブジェクト行は `columns` の順が優先されます．
- table JSON の配列行がヘッダより長い場合，列が自動で増えます．
- table JSON のオブジェクト行に未知のキーがあるとエラーです．配列行の余分な値は許容されます．
- `freezeHeader` と `autoFilter` は `header` が必要です．

### 6.3 数式

**SheetJS 形式**

- `t: "f"` と `f` を使います．`f` に `=` は付けません．
- `f` に `=` を付けると，表示が `==1+1` のようになります．
- `t` を省略して `f` だけを書いた場合も，数式セルとして扱われます．
- `v` にキャッシュ値を持たせることができます．

**table JSON**

- `=` で始まる文字列は数式になります．
- `columns[].formula` に数式テンプレートを書けます．
- `{key}` は現在行のセル，`{key:col}` は列記号，`{r}` は行番号に置換されます．
- 未知のプレースホルダは検証エラーになります．
- `=` で始まる文字列をリテラルにしたい場合は `{"v": "=text"}` を使います．
- クロスシート参照も使えます（例: `=SUM(Sales!D2:D5)`）．

**データ JSON**

- 数式はありません．`=SUM(1,2)` も文字列として書き込まれます．
- `--mode f` や `--mode both` を明示すると，`to-md` / `to-html` は警告を出します．

**`--compute` の共通仕様**

- `v` がない数式セルだけを評価します．`v` がある場合はキャッシュ値を信頼します．
- 数値結果の数式は `f` を保持したまま `v` が補完されます．
- 文字列結果の数式は `t: "s"` になり，`f` は消えます．
- `to-csv` は数式を表示できないため，`--compute` なしでは空欄になり警告が出ます．

### 6.4 スタイルと書式

| 系統 | スタイル参照 | 検証 |
|---|---|---|
| SheetJS | `styles[].id` を `s` で参照 | 未定義 ID は警告．`z` 併用時は書式だけ適用 |
| table JSON | `styles` の名前を参照 | 未定義名・不正な色・不正な罫線名はエラー |
| データ JSON | なし | — |

table JSON の優先順位は，`bodyStyle` / `headerStyle` < `columns[].style` < セルの `style` です．`format` と `align` はその上に適用されます．

スタイルと数値書式が反映されるのは **XLSX のみ**です．

- `to-md` / `to-csv` は値をそのまま表示します．
- `to-html` は配置・フォント・塗り・罫線を反映しますが，数値書式は反映しません．

### 6.5 数値・null・型

| 系統 | 数値のデコード | 大きな整数 |
|---|---|---|
| SheetJS | `encoding/json` の `float64` | 2^53 を超えると精度が落ちる |
| データ JSON | `json.Number` → `int64` / `float64` | パース時点では保持される |
| table JSON | `json.Number` → `int64` / `float64` | パース時点では保持される |

Excel の数値精度は 15 桁程度です．JSON パース時点で保持できても，Excel 上での表示や再計算では丸めが起こり得ます．

null と入れ子オブジェクトの扱い:

- データ JSON の `null` は空欄になります．
- データ JSON の入れ子オブジェクトは，プレビューや CSV では `map[...]` のように表示されます．
- table JSON のセルにオブジェクトを入れると，セルオブジェクトの未知フィールドとしてエラーになります．
- `to-csv --data-json` は行を最大列数に揃えます．`null` は空欄です．

### 6.6 検証

| 系統 | 検証の強さ |
|---|---|
| SheetJS | スキーマはパース失敗時と MCP の validate で使われる．CLI では未知フィールドが無視されることがある |
| データ JSON | 構造の自動判別のみ．内容の意味検証はほぼない |
| table JSON | 未知フィールドを拒否し，シート名・キー重複・スタイル参照・数式プレースホルダを検証する |

スキーマは実行時の挙動に合わせて緩和されています．

- SheetJS は `cells` または `rows` のどちらかがあればよい．
- `cell.t` は任意．`f` があれば数式として扱う．
- table JSON は `rows`，`header`，`columns[].header` のいずれかがあればよい．

### 6.7 往復

`to-json` の出力は常に **book ラッパー + セルマップ**です．

- `rows` は `cells` に展開されます．
- 数式・結合・固定ペイン・オートフィルタ・列幅は保持されます．
- シート順は `book.sheetOrder` で保持されます．
- スタイルは `book.styles` とトップレベル `styles` の両方に出力されます．
- 空シートは `{"name": "Sheet1"}` のように `cells` なしで出力されることがあります．

シート順を保ったまま往復させたい場合は，`to-json` の出力をそのまま `to-xlsx` に戻してください．`book.sheetOrder` を削除すると，シート名のアルファベット順になります．

### 6.8 入力の端

- UTF-8 BOM はすべての入口で除去されます．
- 4 バイト未満の JSON も，`to-md` / `to-html` / `info` で読めるようになりました．
- `[]` はデータ JSON としては空のブックになります．`to-xlsx` と `to-csv --data-json` は「データがない」エラーにします．
- `to-csv` のフラグなし配列は，データ JSON ではなく従来の配列オブジェクト経路です．bool を含む配列はエラーになります．

## 7. 落とし穴

1. **`to-xlsx` / `to-md` / `to-html` / `to-csv` は JSON 形式を自動判定しません**  
   table JSON には `--table-json`，データ JSON には `--data-json` が必要です．`info` と MCP だけが自動判定します．

2. **誤ったフラグを付けると，無言で別物として解釈されることがあります**  
   例: table JSON に `--data-json` を付けると，`tables` が 1 列の配列として扱われます．エラーにならない場合があるため注意してください．

3. **`info` の自動判定は完全ではありません**  
   列名が `rows` や `styles` と衝突する Map-of-Arrays など，曖昧な場合は `--data-json` / `--table-json` を明示してください．

4. **`to-csv` のフラグなし配列はレガシー経路**  
   データ JSON として扱いたい場合は `--data-json` を付けます．

5. **プレビュー系コマンドは書式を反映しません**  
   数値書式やスタイルを確認する場合は XLSX を開いてください．

6. **重複シート名はエラーになりました**  
   以前のように無言で上書きされることはありません．シート名を一意にしてください．

7. **シート順は `book.sheetOrder` で決まります**  
   `book.sheets` のオブジェクト順は保証されません．順序が必要なら `sheetOrder` を指定してください．

8. **警告の終了コードはコマンドによって異なります**  
   `to-xlsx` は変換警告があると exit 1，`to-md` / `to-html` / `to-csv` は警告を出しても exit 0 です．

9. **table JSON の数式はキャッシュ値を持つと再計算されません**  
   `--compute` は `v` がないセルだけを評価します．

## 8. 推奨ワークフロー

1. **一般的な表**  
   table JSON を書き，MCP の `json2xlsx_validate` で検証し，`json2xlsx_preview` で表示を確認してから `to-xlsx` で変換します．

2. **生データ**  
   CSV / SQL / API の結果をデータ JSON として渡し，必要なら `--range` で範囲を絞ります．装飾が必要になった時点で table JSON か SheetJS 形式に移行します．

3. **高度な帳票・往復**  
   SheetJS 形式を使います．既存 XLSX は `to-json` で出力し，`book.sheetOrder` を保持したまま編集して戻します．

4. **形式を確認したい**  
   `info` でシート数・セル数・数式数を確認します．table JSON やデータ JSON も自動判定されます．

## 9. 関連リンク

- [README（日本語）](../README.ja.md)
- [JSON Schema（SheetJS）](../schemas/json2xlsx.schema.json)
- [JSON Schema（table）](../schemas/table.schema.json)
- [sheetjs-json-writer](../skills/sheetjs-json-writer/SKILL.ja.md)
- [table-json-writer](../skills/table-json-writer/SKILL.ja.md)
- [サンプル](../samples/)
