package storage

import (
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
)

type AppStorage struct {
	app        fyne.App
	configPath string
	dbPath     string
	cachePath  string
}

func NewAppStorage(app fyne.App) (*AppStorage, error) {
	var baseDir string
	var err error

	if runtime.GOOS == "android" {
		baseDir = app.Storage().RootURI().Path()
	} else {
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(baseDir, "xengate")
	}

	configPath := filepath.Join(baseDir, "config")
	dbPath := filepath.Join(baseDir, "db")
	cachePath := filepath.Join(baseDir, "cache")

	dirs := []string{configPath, dbPath, cachePath}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	return &AppStorage{
		app:        app,
		configPath: configPath,
		dbPath:     dbPath,
		cachePath:  cachePath,
	}, nil
}

func (s *AppStorage) ConfigPath() string {
	return s.configPath
}

func (s *AppStorage) DBPath() string {
	return s.dbPath
}

func (s *AppStorage) CachePath() string {
	return s.cachePath
}

func (s *AppStorage) EnsureFilePermissions(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// اگر فایل وجود نداشت، یک فایل خالی بساز
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		file.Close()
	}

	return os.Chmod(path, 0o644)
}

func (s *AppStorage) EnsureDirPermissions(dirpath string) error {
	if err := os.MkdirAll(dirpath, 0o755); err != nil {
		return err
	}
	return os.Chmod(dirpath, 0o755)
}

func (s *AppStorage) WriteFile(filepath string, data []byte) error {
	if err := s.EnsureFilePermissions(filepath); err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0o644)
}

func (s *AppStorage) ReadFile(filepath string) ([]byte, error) {
	if err := s.EnsureFilePermissions(filepath); err != nil {
		return nil, err
	}
	return os.ReadFile(filepath)
}

func (s *AppStorage) FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func (s *AppStorage) DeleteFile(filepath string) error {
	return os.Remove(filepath)
}

func (s *AppStorage) CopyFile(src, dst string) error {
	data, err := s.ReadFile(src)
	if err != nil {
		return err
	}
	return s.WriteFile(dst, data)
}

func (s *AppStorage) ListFiles(dirpath string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dirpath, entry.Name()))
		}
	}
	return files, nil
}

func (s *AppStorage) ClearCache() error {
	entries, err := os.ReadDir(s.cachePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(s.cachePath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func (s *AppStorage) GetCacheSize() (int64, error) {
	var size int64
	err := filepath.Walk(s.cachePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
