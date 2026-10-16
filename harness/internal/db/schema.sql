-- Workspace database. Chat tables are the first records. Later config can move here.

CREATE TABLE IF NOT EXISTS chats (
    id TEXT PRIMARY KEY,
    workspace TEXT NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    started_at TEXT NOT NULL,
    ended_at TEXT,
    outcome TEXT,
    reason TEXT,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd REAL NOT NULL DEFAULT 0
);

-- Transcript appended by the loop, including tool results.
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id),
    seq INTEGER NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    tool_call_id TEXT,
    name TEXT,
    tool_calls_json TEXT CHECK (tool_calls_json IS NULL OR json_valid(tool_calls_json)),
    created_at TEXT NOT NULL,
    UNIQUE (chat_id, seq)
);

-- One model call. request_json is the context actually sent.
CREATE TABLE IF NOT EXISTS turns (
    id INTEGER PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id),
    seq INTEGER NOT NULL,
    request_json TEXT NOT NULL CHECK (json_valid(request_json)),
    response_json TEXT NOT NULL CHECK (json_valid(response_json)),
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd REAL NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    prefix_hash TEXT NOT NULL DEFAULT '',
    estimated_input_tokens INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    UNIQUE (chat_id, seq)
);

CREATE TABLE IF NOT EXISTS tool_calls (
    id INTEGER PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id),
    turn_id INTEGER REFERENCES turns(id),
    name TEXT NOT NULL,
    phase TEXT NOT NULL DEFAULT '',
    arguments_json TEXT NOT NULL CHECK (json_valid(arguments_json)),
    result TEXT NOT NULL DEFAULT '',
    ok INTEGER NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

-- Loop facts other than a message or a model call. Kinds are state, compact, loop, verify, and goal.
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id),
    seq INTEGER NOT NULL,
    kind TEXT NOT NULL,
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    created_at TEXT NOT NULL,
    UNIQUE (chat_id, seq)
);

-- Application logs and Go errors tied to a chat session.
CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id),
    seq INTEGER NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    message TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    UNIQUE (chat_id, seq)
);
