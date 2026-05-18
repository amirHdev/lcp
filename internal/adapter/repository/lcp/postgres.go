package lcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strconv"
	"time"

	rootdomain "github.com/amirhdev/ebook-lcp-server/internal/domain"
	userdomain "github.com/amirhdev/ebook-lcp-server/internal/domain"
	domain "github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresPublicationRepository struct {
	db *sql.DB
}

type postgresLicenseRepository struct {
	db *sql.DB
}

type postgresUserRepository struct {
	db *sql.DB
}

type postgresAuditRepository struct {
	db *sql.DB
}

func OpenPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("close rows: %v", err)
			}
		}()
		return nil, err
	}
	return db, nil
}

func EnsurePostgresSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role VARCHAR(32) NOT NULL DEFAULT 'user',
			two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			lcpl TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS publications (
			id VARCHAR(36) PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			title VARCHAR(255) NOT NULL,
			authors JSONB NOT NULL DEFAULT '[]'::jsonb,
			language TEXT NOT NULL DEFAULT '',
			subjects JSONB NOT NULL DEFAULT '[]'::jsonb,
			tags JSONB NOT NULL DEFAULT '[]'::jsonb,
			status TEXT NOT NULL DEFAULT 'active',
			right_print INTEGER,
			right_copy INTEGER,
			file_path TEXT NOT NULL,
			encrypted_path TEXT,
			encrypted_uri TEXT NOT NULL DEFAULT '',
			checksum TEXT NOT NULL DEFAULT '',
			license_duration_days INTEGER NOT NULL DEFAULT 30,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS licenses (
			id VARCHAR(36) PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			publication_id VARCHAR(36) NOT NULL,
			user_id VARCHAR(36) NOT NULL,
			passphrase TEXT NOT NULL,
			hint TEXT NOT NULL,
			publication_url TEXT NOT NULL,
			right_print INTEGER,
			right_copy INTEGER,
			start_date TIMESTAMP,
			end_date TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			lcpl TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS lcp_processes (
			id VARCHAR(36) PRIMARY KEY,
			status VARCHAR(32) NOT NULL,
			publication_id VARCHAR(36),
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (publication_id) REFERENCES publications(id)
		)`,
		`CREATE TABLE IF NOT EXISTS audit_entries (
			id VARCHAR(36) PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			action TEXT NOT NULL,
			actor TEXT NOT NULL,
			resource TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS authors JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS language TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS subjects JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS right_print INTEGER`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS right_copy INTEGER`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS encrypted_uri TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS license_duration_days INTEGER NOT NULL DEFAULT 30`,
		`ALTER TABLE publications ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS lcpl TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default'`,
		`ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_publication_id_fkey`,
		`ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_user_id_fkey`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func NewPostgresPublicationRepository(db *sql.DB) PublicationRepository {
	return &postgresPublicationRepository{db: db}
}

func NewPostgresLicenseRepository(db *sql.DB) LicenseRepository {
	return &postgresLicenseRepository{db: db}
}

func NewPostgresUserRepository(db *sql.DB) userdomain.UserRepository {
	return &postgresUserRepository{db: db}
}

func NewPostgresAuditRepository(db *sql.DB) *postgresAuditRepository {
	return &postgresAuditRepository{db: db}
}

func (r *postgresAuditRepository) Save(ctx context.Context, entry *rootdomain.AuditEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_entries (id, tenant_id, action, actor, resource, resource_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, entry.ID, entry.TenantID, entry.Action, entry.Actor, entry.Resource, entry.ResourceID, entry.CreatedAt)
	return err
}

func (r *postgresAuditRepository) FindRecent(ctx context.Context, limit int) ([]*rootdomain.AuditEntry, error) {
	return r.findRecent(ctx, "", limit)
}

func (r *postgresAuditRepository) FindRecentByTenant(ctx context.Context, tenantID string, limit int) ([]*rootdomain.AuditEntry, error) {
	return r.findRecent(ctx, tenantID, limit)
}

func (r *postgresAuditRepository) findRecent(ctx context.Context, tenantID string, limit int) ([]*rootdomain.AuditEntry, error) {
	query := `
		SELECT id, tenant_id, action, actor, resource, resource_id, created_at
		FROM audit_entries
	`
	args := []any{}
	if tenantID != "" {
		query += ` WHERE tenant_id = $1`
		args = append(args, tenantID)
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT $` + strconv.Itoa(len(args)+1)
		args = append(args, limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*rootdomain.AuditEntry
	for rows.Next() {
		entry := &rootdomain.AuditEntry{}
		if err := rows.Scan(&entry.ID, &entry.TenantID, &entry.Action, &entry.Actor, &entry.Resource, &entry.ResourceID, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *postgresPublicationRepository) Save(ctx context.Context, pub *domain.Publication) error {
	pub.UpdatedAt = time.Now()
	if pub.CreatedAt.IsZero() {
		pub.CreatedAt = pub.UpdatedAt
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO publications (
			id, tenant_id, title, authors, language, subjects, tags, status, right_print, right_copy,
			file_path, encrypted_path, encrypted_uri, checksum, license_duration_days,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4::jsonb, $5, $6::jsonb, $7::jsonb, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			tenant_id = EXCLUDED.tenant_id,
			authors = EXCLUDED.authors,
			language = EXCLUDED.language,
			subjects = EXCLUDED.subjects,
			tags = EXCLUDED.tags,
			status = EXCLUDED.status,
			right_print = EXCLUDED.right_print,
			right_copy = EXCLUDED.right_copy,
			file_path = EXCLUDED.file_path,
			encrypted_path = EXCLUDED.encrypted_path
			, encrypted_uri = EXCLUDED.encrypted_uri,
			checksum = EXCLUDED.checksum,
			license_duration_days = EXCLUDED.license_duration_days,
			updated_at = EXCLUDED.updated_at
	`, pub.ID, pub.TenantID, pub.Title, mustJSON(pub.Authors), pub.Language, mustJSON(pub.Subjects), mustJSON(pub.Tags), pub.Status,
		pub.RightPrint, pub.RightCopy, pub.FilePath, pub.EncryptedPath, pub.EncryptedURI, pub.Checksum, pub.LicenseDurationDays, pub.CreatedAt, pub.UpdatedAt)
	return err
}

func (r *postgresPublicationRepository) FindAll(ctx context.Context) ([]*domain.Publication, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, title, authors, language, subjects, tags, status,
			right_print, right_copy, file_path, encrypted_path, encrypted_uri, checksum, license_duration_days, created_at, updated_at
		FROM publications
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("close rows: %v", err)
		}
	}()

	var pubs []*domain.Publication
	for rows.Next() {
		pub, err := scanPublication(rows)
		if err != nil {
			return nil, err
		}
		pubs = append(pubs, pub)
	}
	return pubs, rows.Err()
}

func (r *postgresPublicationRepository) FindByID(ctx context.Context, id string) (*domain.Publication, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, title, authors, language, subjects, tags, status,
			right_print, right_copy, file_path, encrypted_path, encrypted_uri, checksum, license_duration_days, created_at, updated_at
		FROM publications
		WHERE id = $1
	`, id)
	pub, err := scanPublication(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return pub, err
}

func (r *postgresLicenseRepository) Save(ctx context.Context, license *domain.License) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO licenses (
			id, tenant_id, publication_id, user_id, passphrase, hint, publication_url,
			right_print, right_copy, start_date, end_date, created_at, lcpl
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			passphrase = EXCLUDED.passphrase,
			tenant_id = EXCLUDED.tenant_id,
			hint = EXCLUDED.hint,
			publication_url = EXCLUDED.publication_url,
			right_print = EXCLUDED.right_print,
			right_copy = EXCLUDED.right_copy,
			start_date = EXCLUDED.start_date,
			end_date = EXCLUDED.end_date,
			lcpl = EXCLUDED.lcpl
	`, license.ID, license.TenantID, license.PublicationID, license.UserID, license.Passphrase, license.Hint,
		license.PublicationURL, license.RightPrint, license.RightCopy, license.StartDate,
		license.EndDate, license.CreatedAt, license.LCPL)
	return err
}

func (r *postgresLicenseRepository) FindByID(ctx context.Context, id string) (*domain.License, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, publication_id, user_id, passphrase, hint, publication_url,
			right_print, right_copy, start_date, end_date, created_at, lcpl
		FROM licenses
		WHERE id = $1
	`, id)
	license, err := scanLicense(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return license, err
}

func (r *postgresLicenseRepository) FindByPublication(ctx context.Context, publicationID *string) ([]*domain.License, error) {
	query := `
		SELECT id, tenant_id, publication_id, user_id, passphrase, hint, publication_url,
			right_print, right_copy, start_date, end_date, created_at, lcpl
		FROM licenses
	`
	args := []interface{}{}
	if publicationID != nil {
		query += " WHERE publication_id = $1"
		args = append(args, *publicationID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("close rows: %v", err)
		}
	}()

	var licenses []*domain.License
	for rows.Next() {
		license, err := scanLicense(rows)
		if err != nil {
			return nil, err
		}
		licenses = append(licenses, license)
	}
	return licenses, rows.Err()
}

func (r *postgresLicenseRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM licenses WHERE id = $1", id)
	return err
}

func (r *postgresUserRepository) Ensure(ctx context.Context, user *userdomain.User) error {
	if user == nil || user.ID == "" {
		return nil
	}
	email := user.Email
	if email == "" {
		email = user.ID + "@local"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, role, two_factor_enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			role = EXCLUDED.role
	`, user.ID, email, "managed-by-lcp", "user", false)
	if err != nil {
		return err
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPublication(row rowScanner) (*domain.Publication, error) {
	pub := &domain.Publication{}
	var (
		authors, subjects, tags []byte
		rightPrint, rightCopy   sql.NullInt64
	)
	err := row.Scan(&pub.ID, &pub.TenantID, &pub.Title, &authors, &pub.Language, &subjects, &tags, &pub.Status,
		&rightPrint, &rightCopy, &pub.FilePath, &pub.EncryptedPath, &pub.EncryptedURI, &pub.Checksum, &pub.LicenseDurationDays, &pub.CreatedAt, &pub.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(authors, &pub.Authors)
	_ = json.Unmarshal(subjects, &pub.Subjects)
	_ = json.Unmarshal(tags, &pub.Tags)
	pub.RightPrint = nullIntPtr(rightPrint)
	pub.RightCopy = nullIntPtr(rightCopy)
	if pub.EncryptedURI == "" {
		pub.EncryptedURI = pub.EncryptedPath
	}
	return pub, err
}

func mustJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func scanLicense(row rowScanner) (*domain.License, error) {
	license := &domain.License{}
	var rightPrint, rightCopy sql.NullInt64
	err := row.Scan(&license.ID, &license.TenantID, &license.PublicationID, &license.UserID, &license.Passphrase,
		&license.Hint, &license.PublicationURL, &rightPrint, &rightCopy,
		&license.StartDate, &license.EndDate, &license.CreatedAt, &license.LCPL)
	license.RightPrint = nullIntPtr(rightPrint)
	license.RightCopy = nullIntPtr(rightCopy)
	return license, err
}

func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
