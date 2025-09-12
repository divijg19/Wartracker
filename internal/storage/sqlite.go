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
	if err != nil { return nil, err }
	conn.SetMaxOpenConns(1)
	if err := initSchema(conn); err != nil { return nil, err }
	return &DB{conn: conn}, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS members (
		discord_id TEXT PRIMARY KEY,
		in_game_name TEXT NOT NULL,
		war_orders INTEGER DEFAULT 0,
		lumber INTEGER DEFAULT 0,
		availability TEXT DEFAULT 'Not Set'
	);`)
	return err
}

// UpsertMember inserts or updates member name.
func (d *DB) UpsertMember(ctx context.Context, discordID, inGameName string) error {
	d.mu.Lock(); defer d.mu.Unlock()
	_, err := d.conn.ExecContext(ctx, `INSERT INTO members(discord_id, in_game_name) VALUES(?,?)
		ON CONFLICT(discord_id) DO UPDATE SET in_game_name=excluded.in_game_name`, discordID, inGameName)
	return err
}

func (d *DB) UpdateOrders(ctx context.Context, discordID string, amount int) error {
	d.mu.Lock(); defer d.mu.Unlock()
	res, err := d.conn.ExecContext(ctx, `UPDATE members SET war_orders=? WHERE discord_id=?`, amount, discordID)
	if err != nil { return err }
	if n, _ := res.RowsAffected(); n == 0 { return errors.New("member not registered") }
	return nil
}

func (d *DB) UpdateLumber(ctx context.Context, discordID string, amount int) error {
	d.mu.Lock(); defer d.mu.Unlock()
	res, err := d.conn.ExecContext(ctx, `UPDATE members SET lumber=? WHERE discord_id=?`, amount, discordID)
	if err != nil { return err }
	if n, _ := res.RowsAffected(); n == 0 { return errors.New("member not registered") }
	return nil
}

func (d *DB) UpdateAvailability(ctx context.Context, discordID, slot string) error {
	d.mu.Lock(); defer d.mu.Unlock()
	res, err := d.conn.ExecContext(ctx, `UPDATE members SET availability=? WHERE discord_id=?`, slot, discordID)
	if err != nil { return err }
	if n, _ := res.RowsAffected(); n == 0 { return errors.New("member not registered") }
	return nil
}

func (d *DB) DeleteMember(ctx context.Context, discordID string) error {
	d.mu.Lock(); defer d.mu.Unlock()
	_, err := d.conn.ExecContext(ctx, `DELETE FROM members WHERE discord_id=?`, discordID)
	return err
}

func (d *DB) GetAllMembers(ctx context.Context) ([]Member, error) {
	d.mu.RLock(); defer d.mu.RUnlock()
	rows, err := d.conn.QueryContext(ctx, `SELECT discord_id, in_game_name, war_orders, lumber, availability FROM members ORDER BY in_game_name COLLATE NOCASE`)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.DiscordID, &m.InGameName, &m.WarOrders, &m.Lumber, &m.Availability); err != nil { return nil, err }
		list = append(list, m)
	}
	return list, rows.Err()
}

func (d *DB) EnsureMemberExists(ctx context.Context, discordID string) (bool, error) {
	d.mu.RLock(); defer d.mu.RUnlock()
	var id string
	err := d.conn.QueryRowContext(ctx, `SELECT discord_id FROM members WHERE discord_id=?`, discordID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) { return false, nil }
	return err == nil, err
}

func (d *DB) Close() error { return d.conn.Close() }

// WithTimeout provides standard timeout context for DB operations.
func WithTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 3*time.Second)
}

func FormatMembers(ms []Member, formatter func(m Member) string) string {
	out := ""
	for _, m := range ms { out += formatter(m) + "\n" }
	if out == "" { return "(no members)" }
	return out
}

func (d *DB) DebugPrint(ctx context.Context) {
	members, _ := d.GetAllMembers(ctx)
	fmt.Println("Roster dump:")
	for _, m := range members {
		fmt.Printf("%s => %s Orders:%d Lumber:%d Av:%s\n", m.DiscordID, m.InGameName, m.WarOrders, m.Lumber, m.Availability)
	}
}
