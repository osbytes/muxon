// Package db is the workspace database. Chat transcripts and loop steps are the
// first tables; other saved state can move here later.
package db

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PLACEHOLDER/muxon/harness/internal/security"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

const schemaVersion = 2

// File is the database path for a workspace.
func File(workspace string) string {
	return filepath.Join(workspace, ".muxon", "muxon.sqlite")
}

// DB is the open workspace database.
type DB struct {
	sql *sql.DB
}

// Chat identifies one session row.
type Chat struct {
	ID        string
	Workspace string
	Model     string
	Provider  string
}

// Message is one transcript row.
type Message struct {
	Role       string
	Content    string
	ToolCallID string
	Name       string
	ToolCalls  json.RawMessage
}

// Turn is one model call, including the context that was sent.
type Turn struct {
	Request              json.RawMessage
	Response             json.RawMessage
	InputTokens          int
	OutputTokens         int
	CachedTokens         int
	ReasoningTokens      int
	CostUSD              float64
	Latency              time.Duration
	PrefixHash           string
	EstimatedInputTokens int
}

// ToolCall is one executed or rejected tool call.
type ToolCall struct {
	TurnID    int64
	Name      string
	Phase     string
	Arguments json.RawMessage
	Result    string
	OK        bool
	Error     string
}

// Event is one loop fact that is not a message or a turn.
type Event struct {
	Kind    string
	Payload json.RawMessage
}

// Log is one application log entry tied to a chat session.
type Log struct {
	Level   string
	Message string
	Source  string
	Error   string
}

// Summary closes a chat with the totals from the runs that used it.
type Summary struct {
	Outcome         string
	Reason          string
	InputTokens     int
	OutputTokens    int
	CachedTokens    int
	ReasoningTokens int
	CostUSD         float64
}

// Open creates the workspace database and applies the schema.
func Open(workspace string) (*DB, error) {
	path := File(workspace)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	d := &DB{sql: sqlDB}
	if err := d.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return d, nil
}

// Close closes the database.
func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

func (d *DB) migrate() error {
	if _, err := d.sql.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := d.sql.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return fmt.Errorf("set busy timeout: %w", err)
	}
	var version int
	if err := d.sql.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version >= schemaVersion {
		return nil
	}
	if err := execScript(d.sql, schemaSQL); err != nil {
		return err
	}
	if _, err := d.sql.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}
	return nil
}

// StartChat inserts the chat if this session has not been stored yet.
func (d *DB) StartChat(c Chat) error {
	return d.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO chats (id, workspace, model, provider, started_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(id) DO NOTHING`,
			c.ID, c.Workspace, c.Model, c.Provider, now())
		return err
	})
}

// AppendMessages adds transcript rows in order.
func (d *DB) AppendMessages(chatID string, msgs []Message) error {
	if len(msgs) == 0 {
		return nil
	}
	return d.tx(func(tx *sql.Tx) error {
		seq, err := nextSeq(tx, "messages", chatID)
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			calls := msg.ToolCalls
			if len(calls) == 0 {
				calls = nil
			} else {
				calls = json.RawMessage(redact(string(calls)))
			}
			_, err := tx.Exec(`
				INSERT INTO messages (chat_id, seq, role, content, tool_call_id, name, tool_calls_json, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				chatID, seq, msg.Role, redact(msg.Content), nullIfEmpty(msg.ToolCallID), nullIfEmpty(msg.Name), nullRaw(calls), now())
			if err != nil {
				return err
			}
			seq++
		}
		return nil
	})
}

// RecordTurn stores one model call and returns its row id.
func (d *DB) RecordTurn(chatID string, turn Turn) (int64, error) {
	var id int64
	err := d.tx(func(tx *sql.Tx) error {
		seq, err := nextSeq(tx, "turns", chatID)
		if err != nil {
			return err
		}
		res, err := tx.Exec(`
			INSERT INTO turns (
				chat_id, seq, request_json, response_json,
				input_tokens, output_tokens, cached_tokens, reasoning_tokens,
				cost_usd, latency_ms, prefix_hash, estimated_input_tokens, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			chatID, seq, redact(string(turn.Request)), redact(string(turn.Response)),
			turn.InputTokens, turn.OutputTokens, turn.CachedTokens, turn.ReasoningTokens,
			turn.CostUSD, turn.Latency.Milliseconds(), turn.PrefixHash, turn.EstimatedInputTokens, now())
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	return id, err
}

// RecordTool stores one tool call and its result.
func (d *DB) RecordTool(chatID string, call ToolCall) error {
	args := call.Arguments
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	ok := 0
	if call.OK {
		ok = 1
	}
	var turn any
	if call.TurnID > 0 {
		turn = call.TurnID
	}
	return d.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO tool_calls (chat_id, turn_id, name, phase, arguments_json, result, ok, error, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			chatID, turn, call.Name, call.Phase, redact(string(args)), redact(call.Result), ok, redact(call.Error), now())
		return err
	})
}

// RecordEvent stores one loop fact.
func (d *DB) RecordEvent(chatID string, ev Event) error {
	payload := ev.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	return d.tx(func(tx *sql.Tx) error {
		seq, err := nextSeq(tx, "events", chatID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO events (chat_id, seq, kind, payload_json, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			chatID, seq, ev.Kind, redact(string(payload)), now())
		return err
	})
}

// RecordLog stores one application log entry tied to a chat session.
func (d *DB) RecordLog(chatID string, log Log) error {
	level := log.Level
	if level == "" {
		level = "info"
	}
	return d.tx(func(tx *sql.Tx) error {
		seq, err := nextSeq(tx, "logs", chatID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO logs (chat_id, seq, level, message, source, error, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			chatID, seq, level, redact(log.Message), log.Source, redact(log.Error), now())
		return err
	})
}

// FinishChat adds this run's totals and records how the chat ended.
func (d *DB) FinishChat(chatID string, sum Summary) error {
	return d.tx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE chats SET
				ended_at = ?,
				outcome = ?,
				reason = ?,
				input_tokens = input_tokens + ?,
				output_tokens = output_tokens + ?,
				cached_tokens = cached_tokens + ?,
				reasoning_tokens = reasoning_tokens + ?,
				cost_usd = cost_usd + ?
			WHERE id = ?`,
			now(), sum.Outcome, sum.Reason,
			sum.InputTokens, sum.OutputTokens, sum.CachedTokens, sum.ReasoningTokens, sum.CostUSD,
			chatID)
		return err
	})
}

func execScript(sqlDB *sql.DB, script string) error {
	for _, stmt := range strings.Split(script, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := sqlDB.Exec(stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}
	return nil
}

func (d *DB) tx(fn func(*sql.Tx) error) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func nextSeq(tx *sql.Tx, table, chatID string) (int, error) {
	query := map[string]string{
		"messages": "SELECT COALESCE(MAX(seq), 0) + 1 FROM messages WHERE chat_id = ?",
		"turns":    "SELECT COALESCE(MAX(seq), 0) + 1 FROM turns WHERE chat_id = ?",
		"events":   "SELECT COALESCE(MAX(seq), 0) + 1 FROM events WHERE chat_id = ?",
		"logs":     "SELECT COALESCE(MAX(seq), 0) + 1 FROM logs WHERE chat_id = ?",
	}[table]
	if query == "" {
		return 0, fmt.Errorf("unknown sequence table %q", table)
	}
	var seq int
	err := tx.QueryRow(query, chatID).Scan(&seq)
	return seq, err
}

func redact(s string) string {
	return security.RedactSecrets(s)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullRaw(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
