// Package spanner provides transaction management for Cloud Spanner.
//
// # Transaction Propagation
//
// Write requires a read-write transaction in the context (join-only).
// If no transaction is active, Write returns an error — all writes must
// go through a ReadWriteTransactionScope.
//
// Read and ReadOrSingle use REQUIRED propagation: they join an existing
// transaction if one is active, or create a standalone transaction otherwise.
// Like Write, joining the ambient transaction is the default; the helper names
// describe only the standalone fallback strategy.
//
// Transactions are placed into the context by TransactionScope implementations
// (ReadWriteTransactionScope, ReadOnlyTransactionScope). Application-layer
// command/query handlers call scope.Execute, which starts a transaction and
// embeds it in the context. Repository methods then call Write/Read/
// ReadOrSingle, which transparently join that transaction.
//
// # Choosing a Helper
//
//   - Write:        DML statements (INSERT, UPDATE, DELETE). Requires a
//     read-write transaction in context. Returns an error otherwise.
//   - Read:         The default read helper. Falls back to a ReadOnlyTransaction
//     when standalone, so multiple reads see one consistent snapshot
//     (e.g., COUNT + SELECT). Prefer this when in doubt.
//   - ReadOrSingle: A single read call (ReadRow, Query). Falls back to
//     client.Single() when standalone — cheapest option.
//
// # Enforcement
//
// The spannercheck linter (tools/spannercheck) forbids direct use of
// client.Apply, client.Single, client.ReadOnlyTransaction, and
// client.ReadWriteTransaction in persistence packages, ensuring all
// data access flows through these helpers.
package spanner
