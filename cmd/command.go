package cmd

type Executor interface {
	Call(s Subject, input *Input) string
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
	fn func(s Subject, i *Input) string
}

func Wrap(fn func(s Subject, i *Input) string) Executor {
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
		return false
	}
	return true
}

func (fn *funcExecutor) Call(s Subject, i *Input) string {
	return fn.fn(s, i)
}

func (m *Message) Call(s Subject, i *Input) string {
	return m.Message
}
