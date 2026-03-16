package spanner

import "context"

type Client struct{}
type ReadWriteTransaction struct{}
type ReadOnlyTransaction struct{}
type Mutation struct{}
type Statement struct{}
type Key []interface{}
type KeySet interface{}
type RowIterator struct{}
type Row struct{}

func (c *Client) Apply(ctx context.Context, ms []*Mutation, opts ...interface{}) (interface{}, error) {
	return nil, nil
}
func (c *Client) Single() *ReadOnlyTransaction                    { return nil }
func (c *Client) ReadOnlyTransaction() *ReadOnlyTransaction       { return nil }
func (c *Client) ReadWriteTransaction(ctx context.Context, f func(context.Context, *ReadWriteTransaction) error) (interface{}, error) {
	return nil, nil
}

func (t *ReadWriteTransaction) BufferWrite(ms []*Mutation) error                       { return nil }
func (t *ReadWriteTransaction) Update(ctx context.Context, stmt Statement) (int64, error) {
	return 0, nil
}

func (t *ReadOnlyTransaction) Close() {}
