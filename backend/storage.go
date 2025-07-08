package backend

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
		// در اندروید از مسیر داخلی برنامه استفاده می‌کنیم
		baseDir = app.Storage().RootURI().Path()
	} else {
		// در سیستم‌های دیگر از مسیر کانفیگ کاربر
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(baseDir, "xengate")
	}

	// ساخت مسیرهای مورد نیاز
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
	return filepath.Join(s.dbPath, "xengate.db")
}

func (s *AppStorage) CachePath() string {
	return s.cachePath
}

// برای اطمینان از دسترسی به فایل‌ها در اندروید
func (s *AppStorage) EnsureFilePermissions(filepath string) error {
	// اگر فایل وجود نداشت، یک فایل خالی بساز
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		file, err := os.Create(filepath)
		if err != nil {
			return err
		}
		file.Close()
	}

	// تنظیم دسترسی‌ها
	return os.Chmod(filepath, 0o644)
}

// برای اطمینان از دسترسی به دایرکتوری‌ها در اندروید
func (s *AppStorage) EnsureDirPermissions(dirpath string) error {
	if _, err := os.Stat(dirpath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirpath, 0o755); err != nil {
			return err
		}
	}
	return os.Chmod(dirpath, 0o755)
}

// تابع کمکی برای نوشتن فایل در اندروید
func (s *AppStorage) WriteFile(filepath string, data []byte) error {
	if err := s.EnsureFilePermissions(filepath); err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0o644)
}

// تابع کمکی برای خواندن فایل در اندروید
func (s *AppStorage) ReadFile(filepath string) ([]byte, error) {
	if err := s.EnsureFilePermissions(filepath); err != nil {
		return nil, err
	}
	return os.ReadFile(filepath)
}
