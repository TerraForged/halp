package cmd

type Executor interface {
	Call(input *Input) string
}

type Command struct {
	Exec  Executor
	Fixed bool
	Perms []string
}

type Message struct {
	Message string
}

type funcExecutor struct {
	fn func(i *Input) string
}

func Wrap(fn func(i *Input) string) Executor {
	return &funcExecutor{fn: fn}
}

func (c *Command) RequiresPerm() bool {
	return c.Perms != nil && len(c.Perms) > 0
}

func (c *Command) TestPerm(perm string) bool {
	for _, p := range c.Perms {
		if p == perm {
			return true
		}
	}
	return false
}

func (c *Command) Test(subject Subject) bool {
	if c.RequiresPerm() {
		perms := subject.Perms()
		if perms == nil {
			return false
		}
		for _, perm := range perms {
			if c.TestPerm(perm) {
				return true
			}
		}
	}
	return true
}

func (fn *funcExecutor) Call(i *Input) string {
	return fn.fn(i)
}

func (m *Message) Call(i *Input) string {
	return m.Message
}
