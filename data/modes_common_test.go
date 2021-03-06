package data

import (
	. "gopkg.in/check.v1"
)

var testUserKinds, _ = NewUserModeKinds("(ov)@+")
var testChannelKinds = NewChannelModeKinds("b", "c", "d", "axyz")

func (s *s) TestChannelModeKinds_Create(c *C) {
	m := NewChannelModeKinds("a", "b", "c", "d")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	m = NewChannelModeKinds("a", "b", "c", "d")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	m.Update("d", "c", "b", "a")
	c.Check(m.kinds['d'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['c'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['b'], Equals, ARGS_ONSET)
	c.Check(m.kinds['a'], Equals, ARGS_NONE)
}

func (s *s) TestChannelModeKinds_NewCSV(c *C) {
	m, err := NewChannelModeKindsCSV("")
	c.Check(err, NotNil)

	m, err = NewChannelModeKindsCSV(",,,")
	c.Check(err, IsNil)
	m, err = NewChannelModeKindsCSV(",")
	c.Check(err, NotNil)

	m, err = NewChannelModeKindsCSV("a,b,c,d")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)
}

func (s *s) TestChannelModeKindsUpdate(c *C) {
	m := NewChannelModeKinds("a", "b", "c", "d")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	err := m.UpdateCSV("d,c,b,a")
	c.Check(err, IsNil)
	c.Check(m.kinds['d'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['c'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['b'], Equals, ARGS_ONSET)
	c.Check(m.kinds['a'], Equals, ARGS_NONE)

	m.Update("a", "b", "c", "d")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	err = m.UpdateCSV("")
	c.Check(err, NotNil)
}

func (s *s) TestUserModeKinds_Create(c *C) {
	u, err := NewUserModeKinds("")
	c.Check(u, IsNil)
	c.Check(err, NotNil)
	u, err = NewUserModeKinds("a")
	c.Check(u, IsNil)
	c.Check(err, NotNil)
	u, err = NewUserModeKinds("(a")
	c.Check(u, IsNil)
	c.Check(err, NotNil)

	u, err = NewUserModeKinds("(abcdefghi)!@#$%^&*_")
	c.Check(u, IsNil)
	c.Check(err, NotNil)

	u, err = NewUserModeKinds("(ov)@+")
	c.Check(u, NotNil)
	c.Check(err, IsNil)
	c.Check(u.modeInfo[0], Equals, [2]rune{'o', '@'})
	c.Check(u.modeInfo[1], Equals, [2]rune{'v', '+'})
}

func (s *s) TestUserModeKinds_GetSymbol(c *C) {
	u, err := NewUserModeKinds("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetSymbol('o'), Equals, '@')
	c.Check(u.GetSymbol(' '), Equals, rune(0))
}

func (s *s) TestUserModeKinds_GetMode(c *C) {
	u, err := NewUserModeKinds("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetMode('@'), Equals, 'o')
	c.Check(u.GetMode(' '), Equals, rune(0))
}

func (s *s) TestUserModeKinds_Update(c *C) {
	u, err := NewUserModeKinds("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetModeBit('o'), Not(Equals), 0)
	err = u.UpdateModes("(v)+")
	c.Check(err, IsNil)
	c.Check(u.GetModeBit('o'), Equals, byte(0))

	u, err = NewUserModeKinds("(ov)@+")
	err = u.UpdateModes("")
	c.Check(err, NotNil)
}
