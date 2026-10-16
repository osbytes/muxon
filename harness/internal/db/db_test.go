package db

import (
	"database/sql"
	"testing"
)

func TestOpenAddsLogsOnUpgrade(t *testing.T) {
	dir := t.TempDir()
	database, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.StartChat(Chat{ID: "c1", Workspace: dir, Model: "m", Provider: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	sqlDB, err := sql.Open("sqlite", File(dir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`DROP TABLE logs`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`PRAGMA user_version = 1`); err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.RecordLog("c1", Log{
		Message: "tool call failed",
		Source:  "patch",
		Error:   "patch rejected",
	}); err != nil {
		t.Fatalf("RecordLog() after upgrade: %v", err)
	}

	sqlDB, err = sql.Open("sqlite", File(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	var version int
	if err := sqlDB.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("user_version = %d, want %d", version, schemaVersion)
	}
	var level, message, source, logErr string
	if err := sqlDB.QueryRow(`SELECT level, message, source, error FROM logs WHERE chat_id = ?`, "c1").Scan(&level, &message, &source, &logErr); err != nil {
		t.Fatalf("logs query: %v", err)
	}
	if level != "info" || message != "tool call failed" || source != "patch" || logErr != "patch rejected" {
		t.Fatalf("logs = level %q message %q source %q error %q", level, message, source, logErr)
	}
}
