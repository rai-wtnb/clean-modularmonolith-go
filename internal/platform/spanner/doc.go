// Package spanner provides transaction management for Cloud Spanner.
//
// # Transaction Propagation
//
// All data-access helpers in this package (Write, SingleRead, ConsistentRead)
// use REQUIRED propagation semantics:
//
//   - If a transaction already exists in the context, the helper joins it.
//   - If no transaction exists, the helper creates a new standalone transaction.
//
// Transactions are placed into the context by TransactionScope implementations
// (ReadWriteTransactionScope, ReadOnlyTransactionScope). Application-layer
// command/query handlers call scope.Execute, which starts a transaction and
// embeds it in the context. Repository methods then call Write/SingleRead/
// ConsistentRead, which transparently join that transaction.
//
// When a repository method is called outside a scope (e.g., in tests or
// one-off scripts), the helpers create a short-lived transaction automatically.
//
// # Choosing a Helper
//
//   - Write:          DML statements (INSERT, UPDATE, DELETE). Requires a
//     read-write transaction. Panics if called inside a
//     read-only scope.
//   - SingleRead:     A single read call (ReadRow, Query). Falls back to
//     client.Single() when standalone — cheapest option.
//   - ConsistentRead: Multiple reads that must see the same snapshot
//     (e.g., COUNT + SELECT). Falls back to a ReadOnlyTransaction
//     when standalone.
//
// # Enforcement
//
// The spannercheck linter (tools/spannercheck) forbids direct use of
// client.Apply, client.Single, client.ReadOnlyTransaction, and
// client.ReadWriteTransaction in persistence packages, ensuring all
// data access flows through these helpers.
package spanner
