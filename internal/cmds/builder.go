package cmds

import "sync"

var pool = &sync.Pool{New: newCommandSlice}

// CommandSlice is the command container managed by the sync.Pool
type CommandSlice struct {
	s []string
}

func newCommandSlice() interface{} {
	return &CommandSlice{s: make([]string, 0, 2)}
}

// NewBuilder creates a Builder and initializes the internal sync.Pool
func NewBuilder(initSlot uint16) Builder {
	return Builder{ks: initSlot}
}

// Builder builds commands by reusing CommandSlice from the sync.Pool
type Builder struct {
	ks uint16
}

func get() *CommandSlice {
	return pool.Get().(*CommandSlice)
}

// Put recycles the CommandSlice
func Put(cs *CommandSlice) {
	cs.s = cs.s[:0]
	pool.Put(cs)
}

// Arbitrary allows user to build an arbitrary redis command with Builder.Arbitrary
type Arbitrary Completed

// Arbitrary allows user to build an arbitrary redis command by following Arbitrary.Keys and Arbitrary.Args
func (b Builder) Arbitrary(token ...string) (c Arbitrary) {
	c = Arbitrary{cs: get(), ks: b.ks}
	c.cs.s = append(c.cs.s, token...)
	return c
}

// Keys calculate which key slot the command belongs to.
// Users must use Keys to construct the key part of the command, otherwise
// the command will not be sent to correct redis node.
func (c Arbitrary) Keys(keys ...string) Arbitrary {
	if c.ks != NoSlot {
		for _, k := range keys {
			c.ks = check(c.ks, slot(k))
		}
	}
	c.cs.s = append(c.cs.s, keys...)
	return c
}

// Args is used to construct non-key parts of the command.
func (c Arbitrary) Args(args ...string) Arbitrary {
	c.cs.s = append(c.cs.s, args...)
	return c
}

// Build is used to complete constructing a command
func (c Arbitrary) Build() Completed {
	return Completed(c)
}

// Blocking is used to complete constructing a command and mark it as blocking command.
// Blocking command will occupy a connection from a separated connection pool.
func (c Arbitrary) Blocking() Completed {
	c.cf = blockTag
	return Completed(c)
}

// ReadOnly is used to complete constructing a command and mark it as readonly command.
// ReadOnly will be retried under network issues.
func (c Arbitrary) ReadOnly() Completed {
	c.cf = readonly
	return Completed(c)
}
