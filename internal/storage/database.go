package storage

import (
	"database/sql"
	"path/filepath"

	"xengate/internal/models"
	"xengate/internal/security"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
	cm *security.CryptoManager
}

func InitDatabase(storage *AppStorage, masterPassword string) (*Database, error) {
	dbPath := filepath.Join(storage.DBPath(), "xengate.db")

	// اطمینان از وجود دایرکتوری دیتابیس
	dbDir := filepath.Dir(dbPath)
	if err := storage.EnsureDirPermissions(dbDir); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	cm, err := security.NewCryptoManager(masterPassword)
	if err != nil {
		db.Close()
		return nil, err
	}

	// ایجاد جداول
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Database{db: db, cm: cm}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS connections (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            address TEXT NOT NULL,
            port TEXT NOT NULL,
            type TEXT NOT NULL,
            status INTEGER NOT NULL,
            user TEXT,
            password TEXT,
            proxy_addr TEXT,
            proxy_port INTEGER,
            proxy_mode TEXT,
            connections INTEGER DEFAULT 3,
            max_retries INTEGER DEFAULT 3,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	return err
}

func (db *Database) CreateConnection(conn *models.Connection) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// رمزنگاری پسورد
	var encryptedPassword string
	if conn.Config != nil && conn.Config.Password != "" {
		encryptedPassword, err = db.cm.Encrypt(conn.Config.Password)
		if err != nil {
			return err
		}
	}

	stmt, err := tx.Prepare(`
        INSERT INTO connections (
            name, address, port, type, status, user, password,
            proxy_addr, proxy_port, proxy_mode,
            connections, max_retries
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var proxyAddr, proxyMode string
	var proxyPort, connections, maxRetries int

	if conn.Config != nil {
		proxyAddr = conn.Config.Proxy.ListenAddr
		proxyPort = conn.Config.Proxy.ListenPort
		proxyMode = conn.Config.Proxy.Mode
		connections = conn.Config.Connections
		maxRetries = conn.Config.MaxRetries
	}

	_, err = stmt.Exec(
		conn.Name, conn.Address, conn.Port, conn.Type, conn.Status,
		conn.Config.User, encryptedPassword,
		proxyAddr, proxyPort, proxyMode,
		connections, maxRetries,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *Database) GetConnections() ([]*models.Connection, error) {
	rows, err := db.db.Query(`
        SELECT id, name, address, port, type, status, user, password,
               proxy_addr, proxy_port, proxy_mode,
               connections, max_retries
        FROM connections
        ORDER BY created_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []*models.Connection
	for rows.Next() {
		var id int64
		var conn models.Connection
		var config models.ServerConfig
		var proxyConfig models.ProxyConfig
		var encryptedPassword sql.NullString

		err := rows.Scan(
			&id, &conn.Name, &conn.Address, &conn.Port, &conn.Type, &conn.Status,
			&config.User, &encryptedPassword,
			&proxyConfig.ListenAddr, &proxyConfig.ListenPort,
			&proxyConfig.Mode, &config.Connections, &config.MaxRetries,
		)
		if err != nil {
			return nil, err
		}

		// رمزگشایی پسورد
		if encryptedPassword.Valid {
			password, err := db.cm.Decrypt(encryptedPassword.String)
			if err != nil {
				return nil, err
			}
			config.Password = password
		}

		config.Proxy = proxyConfig
		conn.Config = &config
		conn.ID = id
		connections = append(connections, &conn)
	}

	return connections, nil
}

func (db *Database) UpdateConnection(conn *models.Connection) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var encryptedPassword string
	if conn.Config != nil && conn.Config.Password != "" {
		encryptedPassword, err = db.cm.Encrypt(conn.Config.Password)
		if err != nil {
			return err
		}
	}

	stmt, err := tx.Prepare(`
        UPDATE connections SET
            name = ?, status = ?, user = ?, password = ?,
            proxy_addr = ?, proxy_port = ?, proxy_mode = ?,
            connections = ?, max_retries = ?
        WHERE id = ?
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		conn.Name, conn.Status, conn.Config.User, encryptedPassword,
		conn.Config.Proxy.ListenAddr, conn.Config.Proxy.ListenPort,
		conn.Config.Proxy.Mode, conn.Config.Connections,
		conn.Config.MaxRetries, conn.ID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *Database) DeleteConnection(id int64) error {
	_, err := db.db.Exec("DELETE FROM connections WHERE id = ?", id)
	return err
}

func (db *Database) Close() error {
	return db.db.Close()
}
