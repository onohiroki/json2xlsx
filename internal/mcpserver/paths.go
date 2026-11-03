package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// toolError は MCP のツールエラーを code 付きで表す．
type toolError struct {
	Code string
	Err  error
}

func (e *toolError) Error() string {
	return e.Code + ": " + e.Err.Error()
}

func (e *toolError) Unwrap() error { return e.Err }

func errf(code, format string, args ...any) error {
	return &toolError{Code: code, Err: fmt.Errorf(format, args...)}
}

func (s *server) absPath(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	if len(s.roots) == 0 {
		return filepath.Clean(p)
	}
	return filepath.Join(s.roots[0], p)
}

func isWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return filepath.IsLocal(rel)
}

func (s *server) checkWithin(target string) error {
	for _, root := range s.roots {
		if isWithin(root, target) {
			return nil
		}
	}
	return errf("PATH_NOT_ALLOWED", "%q is outside the allowed directories %v", target, s.roots)
}

// checkReadPath は読み取り可能なファイルパスを解決する．
func (s *server) checkReadPath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errf("INVALID_INPUT", "path must not be empty")
	}
	abs := s.absPath(p)
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", errf("FILE_NOT_FOUND", "%q: %v", p, err)
	}
	if err := s.checkWithin(real); err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", errf("FILE_NOT_FOUND", "%q: %v", p, err)
	}
	if info.IsDir() {
		return "", errf("INVALID_INPUT", "%q is a directory", p)
	}
	return real, nil
}

// checkWritePath は書き込み可能なパスを解決する．
// 親ディレクトリは既存であることを要求する．
func (s *server) checkWritePath(p string, overwrite bool) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errf("INVALID_INPUT", "output_path must not be empty")
	}
	abs := s.absPath(p)
	parent := filepath.Dir(abs)
	parentReal, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", errf("INVALID_INPUT", "output directory %q does not exist: %v", parent, err)
	}
	if err := s.checkWithin(parentReal); err != nil {
		return "", err
	}
	target := filepath.Join(parentReal, filepath.Base(abs))
	if info, err := os.Lstat(target); err == nil {
		if info.IsDir() {
			return "", errf("INVALID_INPUT", "output_path %q is a directory", p)
		}
		if !overwrite {
			return "", errf("OUTPUT_EXISTS", "%q already exists; set overwrite=true to replace it", p)
		}
		if real, err := filepath.EvalSymlinks(target); err == nil {
			if err := s.checkWithin(real); err != nil {
				return "", err
			}
		}
	}
	return target, nil
}

// checkDirPath はディレクトリパスを解決して許可範囲を確認する．
func (s *server) checkDirPath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errf("INVALID_INPUT", "directory must not be empty")
	}
	abs := s.absPath(p)
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", errf("FILE_NOT_FOUND", "%q: %v", p, err)
	}
	if err := s.checkWithin(real); err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", errf("FILE_NOT_FOUND", "%q: %v", p, err)
	}
	if !info.IsDir() {
		return "", errf("INVALID_INPUT", "%q is not a directory", p)
	}
	return real, nil
}

func (s *server) readFile(p string) ([]byte, string, error) {
	real, err := s.checkReadPath(p)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, "", errf("FILE_NOT_FOUND", "%q: %v", p, err)
	}
	if info.Size() > s.opts.MaxInputBytes {
		return nil, "", errf("FILE_TOO_LARGE", "%q is %d bytes; limit is %d bytes", p, info.Size(), s.opts.MaxInputBytes)
	}
	data, err := os.ReadFile(real)
	if err != nil {
		return nil, "", errf("FILE_READ_FAILED", "%q: %v", p, err)
	}
	return data, real, nil
}

func writeFileAtomic(path string, data []byte, overwrite bool) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".json2xlsx-*.tmp")
	if err != nil {
		return errf("FILE_WRITE_FAILED", "create temporary file in %q: %v", dir, err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return errf("FILE_WRITE_FAILED", "write temporary file: %v", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		cleanup()
		return errf("FILE_WRITE_FAILED", "chmod temporary file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return errf("FILE_WRITE_FAILED", "close temporary file: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		if overwrite {
			if rmErr := os.Remove(path); rmErr == nil {
				if err = os.Rename(tmpName, path); err == nil {
					return nil
				}
			}
		}
		_ = os.Remove(tmpName)
		return errf("FILE_WRITE_FAILED", "rename temporary file to %q: %v", path, err)
	}
	return nil
}
