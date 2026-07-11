package json2xlsx

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func extractPicturesFromSheet(f *excelize.File, sheetName string, mode ImageMode, baseDir string) ([]Picture, error) {
	cells, err := f.GetPictureCells(sheetName)
	if err != nil {
		return nil, nil
	}

	imgIndex := 0
	var result []Picture
	for _, cell := range cells {
		pics, err := f.GetPictures(sheetName, cell)
		if err != nil {
			continue
		}
		for _, pic := range pics {
			imgIndex++
			p := Picture{
				Cell: cell,
			}
			if pic.Format != nil {
				p.AltText = pic.Format.AltText
				p.PrintObject = pic.Format.PrintObject
				p.Locked = pic.Format.Locked
				la := pic.Format.LockAspectRatio
				p.LockAspectRatio = &la
				p.OffsetX = pic.Format.OffsetX
				p.OffsetY = pic.Format.OffsetY
				p.ScaleX = pic.Format.ScaleX
				p.ScaleY = pic.Format.ScaleY
				p.Hyperlink = pic.Format.Hyperlink
				p.Positioning = pic.Format.Positioning
			}
			ext := strings.TrimPrefix(pic.Extension, ".")
			if mode == ImageModeFile {
				filename := buildImageFilename(sheetName, imgIndex, ext)
				outPath := filepath.Join(baseDir, filename)
				if err := os.WriteFile(outPath, pic.File, 0644); err != nil {
					return nil, fmt.Errorf("write image file %q: %w", outPath, err)
				}
				p.Path = filename
			} else {
				p.Data = base64.StdEncoding.EncodeToString(pic.File)
				p.Extension = ext
			}
			result = append(result, p)
		}
	}
	return result, nil
}

func extractSheetBackground(f *excelize.File, sheetName string, mode ImageMode, baseDir string) (*SheetBackground, error) {
	sheetIdx, err := f.GetSheetIndex(sheetName)
	if err != nil || sheetIdx < 0 {
		return nil, nil
	}
	wsPath := fmt.Sprintf("xl/worksheets/sheet%d.xml", sheetIdx+1)
	raw, ok := f.Pkg.Load(wsPath)
	if !ok {
		return nil, nil
	}

	var ws struct {
		Picture *struct {
			RID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"http://schemas.openxmlformats.org/spreadsheetml/2006/main picture"`
	}
	dec := xml.NewDecoder(bytes.NewReader(raw.([]byte)))
	if err := dec.Decode(&ws); err != nil {
		return nil, nil
	}
	if ws.Picture == nil || ws.Picture.RID == "" {
		return nil, nil
	}

	relsPath := fmt.Sprintf("xl/worksheets/_rels/sheet%d.xml.rels", sheetIdx+1)
	relsRaw, ok := f.Pkg.Load(relsPath)
	if !ok {
		return nil, nil
	}
	var rels struct {
		Relationships []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
			Type   string `xml:"Type,attr"`
		} `xml:"Relationship"`
	}
	decRels := xml.NewDecoder(bytes.NewReader(relsRaw.([]byte)))
	if err := decRels.Decode(&rels); err != nil {
		return nil, nil
	}

	var mediaPath string
	for _, rel := range rels.Relationships {
		if rel.ID == ws.Picture.RID && strings.Contains(rel.Type, "relationships/image") {
			mediaPath = strings.Replace(rel.Target, "..", "xl", 1)
			break
		}
	}
	if mediaPath == "" {
		return nil, nil
	}

	imgRaw, ok := f.Pkg.Load(mediaPath)
	if !ok {
		return nil, nil
	}
	imgData := imgRaw.([]byte)

	ext := strings.TrimPrefix(filepath.Ext(mediaPath), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	// excelize は .jpeg → .jpg にマップしているため，mediaPath の拡張子が .jpeg のケースも考慮
	if ext == "" {
		ext = "png"
	}

	bg := &SheetBackground{Extension: ext}
	if mode == ImageModeFile {
		filename := buildImageFilename(sheetName, 0, ext)
		outPath := filepath.Join(baseDir, filename)
		if err := os.WriteFile(outPath, imgData, 0644); err != nil {
			return nil, fmt.Errorf("write background image file %q: %w", outPath, err)
		}
		bg.Path = filename
	} else {
		bg.Data = base64.StdEncoding.EncodeToString(imgData)
	}
	return bg, nil
}

func buildImageFilename(sheetName string, index int, ext string) string {
	if ext == "" {
		ext = "png"
	}
	if sheetName != "" {
		safe := strings.Map(func(r rune) rune {
			if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
				return '_'
			}
			return r
		}, sheetName)
		if index == 0 {
			return fmt.Sprintf("%s_background.%s", safe, ext)
		}
		return fmt.Sprintf("%s_%d.%s", safe, index, ext)
	}
	if index == 0 {
		return fmt.Sprintf("Sheet_background.%s", ext)
	}
	return fmt.Sprintf("image_%d.%s", index, ext)
}
