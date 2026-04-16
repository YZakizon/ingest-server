package mocks

import (
    "context"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

type MockTx struct{}

func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) {
    return &MockTx{}, nil
}

func (m *MockTx) Commit(ctx context.Context) error {
    return nil
}

func (m *MockTx) Rollback(ctx context.Context) error {
    return nil
}

func (m *MockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
    return pgconn.NewCommandTag(""), nil
}

func (m *MockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
    return nil, nil
}

func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
    return nil
}

func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
    return nil
}

func (m *MockTx) CopyFrom(ctx context.Context, table pgx.Identifier, columns []string, src pgx.CopyFromSource) (int64, error) {
    return 0, nil
}

func (m *MockTx) Conn() *pgx.Conn {
    return nil
}

func (m *MockTx) LargeObjects() pgx.LargeObjects {
    return pgx.LargeObjects{}
}


func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
    return &pgconn.StatementDescription{}, nil
}
