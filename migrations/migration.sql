CREATE TABLE IF NOT EXISTS runs (
    id UUID PRIMARY KEY,
    trace_id UUID NOT NULL,
    name TEXT NOT NULL,
    batch_key TEXT NOT NULL,
    byte_offset BIGINT NOT NULL,
    byte_length INT NOT NULL
);
