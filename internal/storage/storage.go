package storage

import (
	"io"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

type AppStorage struct {
	app        fyne.App
	configPath string
	dbPath     string
	cachePath  string
}

func NewAppStorage(app fyne.App) (*AppStorage, error) {
	baseDir := app.Storage().RootURI().Path()

	configPath := filepath.Join(baseDir, "config")
	dbPath := filepath.Join(baseDir, "db")
	cachePath := filepath.Join(baseDir, "cache")

	s := &AppStorage{
		app:        app,
		configPath: configPath,
		dbPath:     dbPath,
		cachePath:  cachePath,
	}

	// Ensure directories exist
	dirs := []string{configPath, dbPath, cachePath}
	for _, dir := range dirs {
		if err := s.EnsureDirPermissions(dir); err != nil {
			return nil, err
		}
	}

	return s, nil
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
	if err := s.EnsureDirPermissions(dir); err != nil {
		return err
	}

	uri := storage.NewFileURI(path)
	exists, err := storage.Exists(uri)
	if err != nil {
		return err
	}
	if !exists {
		writer, err := storage.Writer(uri)
		if err != nil {
			return err
		}
		defer writer.Close()
	}
	return nil
}

func (s *AppStorage) EnsureDirPermissions(dirpath string) error {
	uri := storage.NewFileURI(dirpath)
	exists, err := storage.Exists(uri)
	if err != nil {
		return err
	}
	if !exists {
		return storage.CreateListable(uri)
	}
	return nil
}

func (s *AppStorage) WriteFile(filepath string, data []byte) error {
	uri := storage.NewFileURI(filepath)
	writer, err := storage.Writer(uri)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = writer.Write(data)
	return err
}

func (s *AppStorage) ReadFile(filepath string) ([]byte, error) {
	uri := storage.NewFileURI(filepath)
	reader, err := storage.Reader(uri)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (s *AppStorage) FileExists(filepath string) bool {
	uri := storage.NewFileURI(filepath)
	exists, _ := storage.Exists(uri)
	return exists
}

func (s *AppStorage) DeleteFile(filepath string) error {
	uri := storage.NewFileURI(filepath)
	return storage.Delete(uri)
}

func (s *AppStorage) CopyFile(src, dst string) error {
	data, err := s.ReadFile(src)
	if err != nil {
		return err
	}
	return s.WriteFile(dst, data)
}

func (s *AppStorage) ListFiles(dirpath string) ([]string, error) {
	uri := storage.NewFileURI(dirpath)
	list, err := storage.List(uri)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, item := range list {
		_, err := storage.CanList(item)
		if err != nil { // Not a directory
			files = append(files, item.Path())
		}
	}
	return files, nil
}

func (s *AppStorage) ClearCache() error {
	files, err := s.ListFiles(s.cachePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := s.DeleteFile(file); err != nil {
			return err
		}
	}
	return nil
}

func (s *AppStorage) GetCacheSize() (int64, error) {
	var size int64

	files, err := s.ListFiles(s.cachePath)
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		data, err := s.ReadFile(file)
		if err != nil {
			continue
		}
		size += int64(len(data))
	}
	return size, nil
}
