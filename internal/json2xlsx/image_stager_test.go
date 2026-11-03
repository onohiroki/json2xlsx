package json2xlsx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestStagedImagesConflict(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "Sheet1_1.png")
	old := []byte("old image")
	if err := os.WriteFile(final, old, 0o644); err != nil {
		t.Fatal(err)
	}

	stager, err := newStagedImages(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	err = stager.Stage("Sheet1_1.png", []byte("new image"))
	var ie *ImageExistsError
	if !errors.As(err, &ie) {
		t.Fatalf("Stage error = %v, want ImageExistsError", err)
	}
	_ = stager.Cleanup()

	got, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatalf("existing image was modified: %q", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".json2xlsx-tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("temp directories left behind: %v", matches)
	}
}

func TestStagedImagesCommitOverwriteFinalize(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "Sheet1_1.png")
	if err := os.WriteFile(final, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	stager, err := newStagedImages(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := stager.Stage("Sheet1_1.png", []byte("new")); err != nil {
		t.Fatal(err)
	}
	if err := stager.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := stager.Finalize(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("image = %q, want new", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".json2xlsx-tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("temp directories left behind: %v", matches)
	}
}

func TestStagedImagesRollbackRestoresBackup(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "Sheet1_1.png")
	if err := os.WriteFile(final, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	stager, err := newStagedImages(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := stager.Stage("Sheet1_1.png", []byte("new")); err != nil {
		t.Fatal(err)
	}
	if err := stager.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := stager.Rollback(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("image = %q, want old", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".json2xlsx-tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("temp directories left behind: %v", matches)
	}
}

func TestStagedImagesConflictAfterStaging(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "Sheet1_1.png")
	second := filepath.Join(dir, "Sheet1_2.png")
	if err := os.WriteFile(second, []byte("old second"), 0o644); err != nil {
		t.Fatal(err)
	}

	stager, err := newStagedImages(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := stager.Stage("Sheet1_1.png", []byte("new first")); err != nil {
		t.Fatal(err)
	}
	err = stager.Stage("Sheet1_2.png", []byte("new second"))
	var ie *ImageExistsError
	if !errors.As(err, &ie) {
		t.Fatalf("Stage error = %v, want ImageExistsError", err)
	}
	if err := stager.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatalf("first image should not have been committed: %v", err)
	}
	got, _ := os.ReadFile(second)
	if string(got) != "old second" {
		t.Fatalf("second image = %q", got)
	}
}

func TestStagedImagesDuplicateName(t *testing.T) {
	dir := t.TempDir()
	stager, err := newStagedImages(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := stager.Stage("Sheet1_1.png", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := stager.Stage("Sheet1_1.png", []byte("b")); err == nil {
		t.Fatal("expected duplicate filename error")
	}
	_ = stager.Cleanup()
}

func testXLSXWithPicture(t *testing.T, dir string) []byte {
	t.Helper()
	img, err := os.ReadFile("../../samples/sample_image.png")
	if err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	if err := f.AddPictureFromBytes("Sheet1", "A1", &excelize.Picture{Extension: ".png", File: img}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return buf.Bytes()
}

func TestToJSONStagedImageConflict(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Sheet1_1.png"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	input := testXLSXWithPicture(t, dir)

	var out bytes.Buffer
	_, err := ToJSONWithOptionsStaged(bytes.NewReader(input), &out, ToJSONOptions{
		ImageMode: ImageModeFile,
		BaseDir:   dir,
	})
	var ie *ImageExistsError
	if !errors.As(err, &ie) {
		t.Fatalf("error = %v, want ImageExistsError", err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "Sheet1_1.png"))
	if string(got) != "old" {
		t.Fatalf("image = %q, want old", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".json2xlsx-tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("temp directories left behind: %v", matches)
	}
}

func TestToJSONStagedImageOverwrite(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "Sheet1_1.png")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	input := testXLSXWithPicture(t, dir)

	var out bytes.Buffer
	stager, err := ToJSONWithOptionsStaged(bytes.NewReader(input), &out, ToJSONOptions{
		ImageMode:       ImageModeFile,
		BaseDir:         dir,
		OverwriteImages: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stager == nil {
		t.Fatal("stager is nil")
	}
	defer stager.Cleanup()
	if err := stager.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := stager.Finalize(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(oldPath)
	if string(got) == "old" || len(got) == 0 {
		t.Fatalf("image was not replaced: %q", got)
	}
	if !strings.Contains(out.String(), `"path": "Sheet1_1.png"`) {
		t.Fatalf("JSON does not reference the image: %s", out.String())
	}
}

func TestToJSONStagedBackgroundConflict(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Sheet1_background.png"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	img, err := os.ReadFile("../../samples/sample_image.png")
	if err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	if err := f.SetSheetBackgroundFromBytes("Sheet1", ".png", img); err != nil {
		t.Fatal(err)
	}
	var xlsx bytes.Buffer
	if err := f.Write(&xlsx); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	var out bytes.Buffer
	_, err = ToJSONWithOptionsStaged(bytes.NewReader(xlsx.Bytes()), &out, ToJSONOptions{
		ImageMode: ImageModeFile,
		BaseDir:   dir,
	})
	var ie *ImageExistsError
	if !errors.As(err, &ie) {
		t.Fatalf("error = %v, want ImageExistsError", err)
	}
}
