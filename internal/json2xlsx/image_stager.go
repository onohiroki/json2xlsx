package json2xlsx

import (
	"fmt"
	"os"
	"path/filepath"
)

// ImageExistsError は image_mode=file の出力先に既存ファイルがあることを表す．
type ImageExistsError struct {
	Path string
}

func (e *ImageExistsError) Error() string {
	return fmt.Sprintf("image file %q already exists", e.Path)
}

type stagedImage struct {
	name      string
	tempPath  string
	finalPath string
}

type imageBackup struct {
	finalPath  string
	backupPath string
}

// StagedImages は image_mode=file の画像を一時ディレクトリに書き出し，
// JSON の書き込みが成功した後に最終ファイルへ反映する．
type StagedImages struct {
	baseDir   string
	tmpDir    string
	overwrite bool
	files     []stagedImage
	backups   []imageBackup
	committed bool
	finalized bool
}

func newStagedImages(baseDir string, overwrite bool) (*StagedImages, error) {
	if baseDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		baseDir = cwd
	}
	info, err := os.Stat(baseDir)
	if err != nil {
		return nil, fmt.Errorf("image output directory %q: %w", baseDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("image output path %q is not a directory", baseDir)
	}
	return &StagedImages{baseDir: baseDir, overwrite: overwrite}, nil
}

// Stage は画像を一時ディレクトリに書き出す．
// overwrite=false で最終ファイルが存在する場合は ImageExistsError を返す．
func (s *StagedImages) Stage(name string, data []byte) error {
	if s == nil {
		return fmt.Errorf("staged images is nil")
	}
	if name == "" || filepath.Base(name) != name {
		return fmt.Errorf("invalid image filename %q", name)
	}
	finalPath := filepath.Join(s.baseDir, name)
	for _, f := range s.files {
		if f.finalPath == finalPath {
			return fmt.Errorf("duplicate image filename %q", name)
		}
	}
	info, err := os.Lstat(finalPath)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("image target %q is a directory", finalPath)
		}
		if !s.overwrite {
			return &ImageExistsError{Path: finalPath}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check image target %q: %w", finalPath, err)
	}

	if s.tmpDir == "" {
		tmpDir, err := os.MkdirTemp(s.baseDir, ".json2xlsx-tmp-*")
		if err != nil {
			return fmt.Errorf("create image temp directory in %q: %w", s.baseDir, err)
		}
		s.tmpDir = tmpDir
	}
	tempPath := filepath.Join(s.tmpDir, name)
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("write staged image %q: %w", tempPath, err)
	}
	s.files = append(s.files, stagedImage{name: name, tempPath: tempPath, finalPath: finalPath})
	return nil
}

// Commit はステージ済み画像を最終パスへ移動する．
// overwrite=true の場合は既存ファイルをバックアップし，失敗時に復元できるようにする．
func (s *StagedImages) Commit() error {
	if s == nil || len(s.files) == 0 {
		return nil
	}
	if s.committed {
		return nil
	}

	for _, f := range s.files {
		info, err := os.Lstat(f.finalPath)
		if err == nil {
			if info.IsDir() {
				return fmt.Errorf("image target %q is a directory", f.finalPath)
			}
			if !s.overwrite {
				return &ImageExistsError{Path: f.finalPath}
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check image target %q: %w", f.finalPath, err)
		}
	}

	if s.overwrite {
		backupDir := filepath.Join(s.tmpDir, "backup")
		if err := os.MkdirAll(backupDir, 0o700); err != nil {
			return fmt.Errorf("create image backup directory: %w", err)
		}
		for _, f := range s.files {
			if _, err := os.Lstat(f.finalPath); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				s.restoreBackups()
				return fmt.Errorf("check image target %q: %w", f.finalPath, err)
			}
			backupPath := filepath.Join(backupDir, f.name)
			if err := os.Rename(f.finalPath, backupPath); err != nil {
				s.restoreBackups()
				return fmt.Errorf("backup existing image %q: %w", f.finalPath, err)
			}
			s.backups = append(s.backups, imageBackup{finalPath: f.finalPath, backupPath: backupPath})
		}
	}

	moved := 0
	for _, f := range s.files {
		if err := os.Rename(f.tempPath, f.finalPath); err != nil {
			for i := 0; i < moved; i++ {
				_ = os.Remove(s.files[i].finalPath)
			}
			s.restoreBackups()
			return fmt.Errorf("commit image %q: %w", f.finalPath, err)
		}
		moved++
	}
	s.committed = true
	return nil
}

// Rollback は Commit で移動した画像を削除し，バックアップを復元する．
func (s *StagedImages) Rollback() error {
	if s == nil {
		return nil
	}
	var firstErr error
	if s.committed {
		for _, f := range s.files {
			if err := os.Remove(f.finalPath); err != nil && !os.IsNotExist(err) && firstErr == nil {
				firstErr = err
			}
		}
	}
	s.committed = false
	if err := s.restoreBackups(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := s.cleanupDir(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// Finalize はバックアップと一時ディレクトリを削除する．
func (s *StagedImages) Finalize() error {
	if s == nil {
		return nil
	}
	s.backups = nil
	s.finalized = true
	return s.cleanupDir()
}

// Cleanup は一時ディレクトリを削除する．Commit 後に呼ぶ場合は Finalize か Rollback を先に呼ぶ．
func (s *StagedImages) Cleanup() error {
	if s == nil {
		return nil
	}
	return s.cleanupDir()
}

func (s *StagedImages) cleanupDir() error {
	if s.tmpDir == "" {
		return nil
	}
	err := os.RemoveAll(s.tmpDir)
	if err == nil {
		s.tmpDir = ""
	}
	return err
}

func (s *StagedImages) restoreBackups() error {
	var firstErr error
	for i := len(s.backups) - 1; i >= 0; i-- {
		b := s.backups[i]
		if err := os.Remove(b.finalPath); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
		if err := os.Rename(b.backupPath, b.finalPath); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.backups = nil
	return firstErr
}
