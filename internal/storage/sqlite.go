package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps sql.DB for concurrency control.
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// NewConnection initializes the SQLite database and creates schema if missing.
func NewConnection(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	if err := initSchema(conn); err != nil {
		return nil, err
	}
	// Best-effort WAL for robustness and online backups
	_, _ = conn.Exec(`PRAGMA journal_mode=WAL;`)
	return &DB{conn: conn}, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS members (
		discord_id TEXT PRIMARY KEY,
		in_game_name TEXT NOT NULL,
		war_orders INTEGER DEFAULT 0,
		lumber INTEGER DEFAULT 0,
	availability TEXT DEFAULT 'Not Set',
	guild_role_id TEXT DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS leader (
		id INTEGER PRIMARY KEY CHECK (id=1),
		owner TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);`)
	if err != nil {
		return err
	}
	// Ensure column exists for older databases
	return ensureGuildRoleColumn(db)
}

func ensureGuildRoleColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(members);`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "guild_role_id" {
			return nil
		}
	}
	_, err = db.Exec(`ALTER TABLE members ADD COLUMN guild_role_id TEXT DEFAULT ''`)
	return err
}

// UpsertMember inserts or updates member name.
func (d *DB) UpsertMember(ctx context.Context, discordID, inGameName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.ExecContext(ctx, `INSERT INTO members(discord_id, in_game_name) VALUES(?,?)
		ON CONFLICT(discord_id) DO UPDATE SET in_game_name=excluded.in_game_name`, discordID, inGameName)
	return err
}

func (d *DB) UpdateOrders(ctx context.Context, discordID string, amount int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.conn.ExecContext(ctx, `UPDATE members SET war_orders=? WHERE discord_id=?`, amount, discordID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("member not registered")
	}
	return nil
}

func (d *DB) UpdateLumber(ctx context.Context, discordID string, amount int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.conn.ExecContext(ctx, `UPDATE members SET lumber=? WHERE discord_id=?`, amount, discordID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("member not registered")
	}
	return nil
}

// InsertMemberIfMissing inserts a new member with name if not present; existing records are left unchanged.
func (d *DB) InsertMemberIfMissing(ctx context.Context, discordID, inGameName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.ExecContext(ctx, `INSERT OR IGNORE INTO members(discord_id, in_game_name) VALUES(?,?)`, discordID, inGameName)
	return err
}

func (d *DB) UpdateAvailability(ctx context.Context, discordID, slot string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.conn.ExecContext(ctx, `UPDATE members SET availability=? WHERE discord_id=?`, slot, discordID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("member not registered")
	}
	return nil
}

func (d *DB) DeleteMember(ctx context.Context, discordID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.ExecContext(ctx, `DELETE FROM members WHERE discord_id=?`, discordID)
	return err
}

func (d *DB) GetAllMembers(ctx context.Context) ([]Member, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.conn.QueryContext(ctx, `SELECT discord_id, in_game_name, war_orders, lumber, availability, guild_role_id FROM members ORDER BY in_game_name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.DiscordID, &m.InGameName, &m.WarOrders, &m.Lumber, &m.Availability, &m.GuildRoleID); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// UpdateMemberRole sets the member's guild role id.
func (d *DB) UpdateMemberRole(ctx context.Context, discordID, roleID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.conn.ExecContext(ctx, `UPDATE members SET guild_role_id=? WHERE discord_id=?`, roleID, discordID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("member not registered")
	}
	return nil
}

func (d *DB) EnsureMemberExists(ctx context.Context, discordID string) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var id string
	err := d.conn.QueryRowContext(ctx, `SELECT discord_id FROM members WHERE discord_id=?`, discordID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (d *DB) Close() error { return d.conn.Close() }

// WithTimeout provides standard timeout context for DB operations.
func WithTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 3*time.Second)
}

func FormatMembers(ms []Member, formatter func(m Member) string) string {
	out := ""
	for _, m := range ms {
		out += formatter(m) + "\n"
	}
	if out == "" {
		return "(no members)"
	}
	return out
}

func (d *DB) DebugPrint(ctx context.Context) {
	members, _ := d.GetAllMembers(ctx)
	fmt.Println("Roster dump:")
	for _, m := range members {
		fmt.Printf("%s => %s Orders:%d Lumber:%d Av:%s\n", m.DiscordID, m.InGameName, m.WarOrders, m.Lumber, m.Availability)
	}
}

// Leader election (active/standby) helpers for zero-downtime cutover

// TryAcquireLeader attempts to acquire or renew leadership with a lease.
// If takeoverIfStale is true, it will replace a stale owner (updated_at older than lease).
func (d *DB) TryAcquireLeader(ctx context.Context, instanceID string, lease time.Duration, takeoverIfStale bool) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().Unix()
	staleBefore := time.Now().Add(-lease).Unix()

	var owner string
	var updated int64
	err = tx.QueryRow(`SELECT owner, updated_at FROM leader WHERE id=1`).Scan(&owner, &updated)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.Exec(`INSERT INTO leader(id, owner, updated_at) VALUES(1, ?, ?)`, instanceID, now); err != nil {
			return false, err
		}
	case err == nil:
		if owner == instanceID {
			if _, err := tx.Exec(`UPDATE leader SET updated_at=? WHERE id=1 AND owner=?`, now, instanceID); err != nil {
				return false, err
			}
		} else {
			if updated <= staleBefore && takeoverIfStale {
				if _, err := tx.Exec(`UPDATE leader SET owner=?, updated_at=? WHERE id=1`, instanceID, now); err != nil {
					return false, err
				}
			} else {
				return false, tx.Commit()
			}
		}
	default:
		return false, err
	}

	return true, tx.Commit()
}

// RenewLeader refreshes the lease timestamp for the current owner.
func (d *DB) RenewLeader(ctx context.Context, instanceID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.ExecContext(ctx, `UPDATE leader SET updated_at=? WHERE id=1 AND owner=?`, time.Now().Unix(), instanceID)
	return err
}

// ReleaseLeader relinquishes leadership if owned by instanceID.
func (d *DB) ReleaseLeader(ctx context.Context, instanceID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.ExecContext(ctx, `DELETE FROM leader WHERE id=1 AND owner=?`, instanceID)
	return err
}
