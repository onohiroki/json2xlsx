package json2xlsx

import (
	"github.com/xuri/excelize/v2"
)

// toExcelizeSparkline は Sparkline を excelize.SparklineOptions に変換する．
// Location と Range は単一文字列から []string に変換する（excelize は配列だが，
// json2xlsx では 1 スパークライン = 1 pair とする）．
// 色フィールドには stripHash を適用する．
func toExcelizeSparkline(s Sparkline) *excelize.SparklineOptions {
	return &excelize.SparklineOptions{
		Location:      []string{s.Location},
		Range:         []string{s.Range},
		Type:          s.Type,
		Weight:        s.Weight,
		DateAxis:      s.DateAxis,
		Markers:       s.Markers,
		High:          s.High,
		Low:           s.Low,
		First:         s.First,
		Last:          s.Last,
		Negative:      s.Negative,
		Axis:          s.Axis,
		Hidden:        s.Hidden,
		Reverse:       s.Reverse,
		Style:         s.Style,
		SeriesColor:   stripHash(s.SeriesColor),
		NegativeColor: stripHash(s.NegativeColor),
		MarkersColor:  stripHash(s.MarkersColor),
		FirstColor:    stripHash(s.FirstColor),
		LastColor:     stripHash(s.LastColor),
		HightColor:    stripHash(s.HighColor),
		LowColor:      stripHash(s.LowColor),
		EmptyCells:    s.EmptyCells,
		Max:           s.Max,
		CustMax:       s.CustMax,
		Min:           s.Min,
		CustMin:       s.CustMin,
	}
}
