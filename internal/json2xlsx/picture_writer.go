package json2xlsx

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/xuri/excelize/v2"
)

func resolvePath(path, baseDir string) string {
	// If the path is absolute or baseDir is empty, return as-is.
	if filepath.IsAbs(path) || baseDir == "" {
		return path
	}
	// Avoid double‑prefixing when the relative path already starts with the base directory.
	cleanPath := filepath.Clean(path)
	baseClean := filepath.Clean(baseDir)
	// If the relative path starts with the directory name of baseDir (e.g., "samples/sample_image.png" when baseDir is "/.../samples"), strip that leading component.
	dirName := filepath.Base(baseClean)
	if strings.HasPrefix(cleanPath, dirName+string(os.PathSeparator)) {
		cleanPath = strings.TrimPrefix(cleanPath, dirName+string(os.PathSeparator))
	}
	return filepath.Join(baseDir, cleanPath)
}

func addPicturesToSheet(f *excelize.File, sheet string, pictures []Picture, baseDir string) error {
	for _, pic := range pictures {
		if pic.Cell == "" {
			continue
		}
		opts := &excelize.GraphicOptions{
			AltText:         pic.AltText,
			PrintObject:     pic.PrintObject,
			Locked:          pic.Locked,
			LockAspectRatio: pic.LockAspectRatio != nil && *pic.LockAspectRatio,
			OffsetX:         pic.OffsetX,
			OffsetY:         pic.OffsetY,
			ScaleX:          pic.ScaleX,
			ScaleY:          pic.ScaleY,
			Hyperlink:       pic.Hyperlink,
			Positioning:     pic.Positioning,
		}
		if pic.Path != "" {
			resolved := resolvePath(pic.Path, baseDir)
			if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
				if err := f.AddPicture(sheet, pic.Cell, resolved, opts); err != nil {
					return fmt.Errorf("add picture %q: %w", pic.Path, err)
				}
			} else {
				return fmt.Errorf("picture path %q resolved to %q: %w", pic.Path, resolved, err)
			}
		} else if pic.Data != "" {
			data, err := base64.StdEncoding.DecodeString(pic.Data)
			if err != nil {
				return fmt.Errorf("decode base64 picture data: %w", err)
			}
			ext := "." + pic.Extension
			if ext == "." {
				ext = ".png"
			}
			if err := f.AddPictureFromBytes(sheet, pic.Cell, &excelize.Picture{
				Extension: ext,
				File:      data,
				Format:    opts,
			}); err != nil {
				return fmt.Errorf("add picture from bytes: %w", err)
			}
		}
	}
	return nil
}

func setSheetBackgroundImage(f *excelize.File, sheet string, bg *SheetBackground, baseDir string) error {
	if bg == nil {
		return nil
	}
	if bg.Path != "" {
		resolved := resolvePath(bg.Path, baseDir)
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			if err := f.SetSheetBackground(sheet, resolved); err != nil {
				return fmt.Errorf("set sheet background %q: %w", bg.Path, err)
			}
			return nil
		}
		return fmt.Errorf("background path %q resolved to %q: %w", bg.Path, resolved, os.ErrNotExist)
	}
	if bg.Data != "" {
		data, err := base64.StdEncoding.DecodeString(bg.Data)
		if err != nil {
			return fmt.Errorf("decode base64 background data: %w", err)
		}
		ext := "." + bg.Extension
		if ext == "." {
			ext = ".png"
		}
		if err := f.SetSheetBackgroundFromBytes(sheet, ext, data); err != nil {
			return fmt.Errorf("set sheet background from bytes: %w", err)
		}
		return nil
	}
	return nil
}
