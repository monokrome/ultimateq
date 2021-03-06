package data

import (
	"strings"
	"testing"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/aarondl/ultimateq/irc"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

func init() {
	// Speed up bcrypt for tests.
	StoredUserPwdCost = bcrypt.MinCost
	// Invalidate the Store cache enough to be testable.
	nMaxCache = 1
}

var (
	uname    = "user"
	password = "pass"
	host     = `nick!user@host`
	network  = "irc.network.net"
	users    = []string{"nick1!user1@host1", "nick2!user2@host2"}
	nicks    = []string{"nick1", "nick2"}
	channels = []string{"#CHAN1", "#CHAN2"}
	channel  = "#CHAN1"

	self = Self{
		User: NewUser("me!my@host.com"),
	}

	netInfo = irc.NewNetworkInfo()
)

func (s *s) TestState(c *C) {
	st, err := NewState(netInfo)
	c.Check(st, NotNil)
	c.Check(err, IsNil)
	c.Check(st.Self.ChannelModes, NotNil)

	st, err = NewState(nil)
	c.Check(err, Equals, errProtoCapsMissing)

	// Should die on creating kinds
	fakeCaps := &irc.NetworkInfo{}
	fakeCaps.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=#&", "PREFIX=(ov)@+",
	}})
	st, err = NewState(fakeCaps)
	c.Check(st, IsNil)
	c.Check(err, NotNil)

	// Should die on creating user modes
	fakeCaps = &irc.NetworkInfo{}
	fakeCaps.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=#&", "CHANMODES=a,b,c,d",
	}})
	st, err = NewState(fakeCaps)
	c.Check(st, IsNil)
	c.Check(err, NotNil)
}

func (s *s) TestState_UpdateProtoCaps(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)

	fakeNetInfo := &irc.NetworkInfo{}
	fakeNetInfo.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=!", "PREFIX=(q)@", "CHANMODES=,,,q",
	}})
	fakeNetInfo.ParseMyInfo(&irc.Event{Args: []string{
		"nick", "irc.test.net", "test-12", "q", "abc",
	}})

	c.Assert(st.kinds.kinds['q'], Equals, 0)
	c.Assert(st.umodes.GetModeBit('q'), Equals, byte(0))
	st.SetNetworkInfo(fakeNetInfo)
	c.Assert(st.kinds.kinds['q'], Not(Equals), 0)
	c.Assert(st.umodes.GetModeBit('q'), Not(Equals), 0)
}

func (s *s) TestState_GetUser(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.GetUser(users[1]), IsNil)
	st.addUser(users[0])
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUser(users[1]), IsNil)
	st.addUser(users[1])
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUser(users[1]), NotNil)

	st, err = NewState(netInfo)
	c.Check(err, IsNil)
	oldHost := "nick!user@host.com"
	newHost := "nick!user@host.net"
	st.addUser(oldHost)
	c.Check(st.GetUser(oldHost).Host(), Equals, oldHost)
	c.Check(st.GetUser(newHost).Host(), Not(Equals), newHost)
	st.addUser(newHost)
	c.Check(st.GetUser(oldHost).Host(), Not(Equals), oldHost)
	c.Check(st.GetUser(newHost).Host(), Equals, newHost)
}

func (s *s) TestState_GetChannel(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetChannel(channels[0]), IsNil)
	c.Check(st.GetChannel(channels[1]), IsNil)
	st.addChannel(channels[0])
	c.Check(st.GetChannel(channels[0]), NotNil)
	c.Check(st.GetChannel(channels[1]), IsNil)
	st.addChannel(channels[1])
	c.Check(st.GetChannel(channels[0]), NotNil)
	c.Check(st.GetChannel(channels[1]), NotNil)
}

func (s *s) TestState_GetUsersChannelModes(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	st.addUser(users[0])
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), IsNil)
	st.addChannel(channels[0])
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), IsNil)

	st.addToChannel(users[0], channels[0])
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), NotNil)
}

func (s *s) TestState_GetNUsers(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetNUsers(), Equals, 0)
	st.addUser(users[0])
	st.addUser(users[0]) // Test that adding a user twice does nothing.
	c.Check(st.GetNUsers(), Equals, 1)
	st.addUser(users[1])
	c.Check(st.GetNUsers(), Equals, 2)
}

func (s *s) TestState_GetNChannels(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetNChannels(), Equals, 0)
	st.addChannel(channels[0])
	st.addChannel(channels[0]) // Test that adding a channel twice does nothing.
	c.Check(st.GetNChannels(), Equals, 1)
	st.addChannel(channels[1])
	c.Check(st.GetNChannels(), Equals, 2)
}

func (s *s) TestState_GetNUserChans(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetNUserChans(users[0]), Equals, 0)
	c.Check(st.GetNUserChans(users[0]), Equals, 0)
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	c.Check(st.GetNUserChans(users[0]), Equals, 0)
	c.Check(st.GetNUserChans(users[0]), Equals, 0)
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[0]) // Test no duplicate adds.
	st.addToChannel(users[0], channels[1])
	st.addToChannel(users[1], channels[0])
	c.Check(st.GetNUserChans(users[0]), Equals, 2)
	c.Check(st.GetNUserChans(users[1]), Equals, 1)
}

func (s *s) TestState_GetNChanUsers(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetNChanUsers(channels[0]), Equals, 0)
	c.Check(st.GetNChanUsers(channels[0]), Equals, 0)
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	c.Check(st.GetNChanUsers(channels[0]), Equals, 0)
	c.Check(st.GetNChanUsers(channels[0]), Equals, 0)
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	st.addToChannel(users[1], channels[0])
	c.Check(st.GetNChanUsers(channels[0]), Equals, 2)
	c.Check(st.GetNChanUsers(channels[1]), Equals, 1)
}

func (s *s) TestState_EachUser(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	st.addUser(users[0])
	st.addUser(users[1])
	i := 0
	st.EachUser(func(u *User) {
		has := false
		for _, user := range users {
			if user == u.Host() {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
		i++
	})
	c.Check(i, Equals, 2)
}

func (s *s) TestState_EachChannel(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	i := 0
	st.EachChannel(func(ch *Channel) {
		has := false
		for _, channel := range channels {
			if channel == ch.String() {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
		i++
	})
	c.Check(i, Equals, 2)
}

func (s *s) TestState_EachUserChan(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	i := 0
	st.EachUserChan(users[0], func(uc *UserChannel) {
		has := false
		for _, channel := range channels {
			if channel == uc.Channel.String() {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
		i++
	})
	c.Check(i, Equals, 2)
}

func (s *s) TestState_EachChanUser(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	i := 0
	st.EachChanUser(channels[0], func(cu *ChannelUser) {
		has := false
		for _, user := range users {
			if user == cu.User.Host() {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
		i++
	})
	c.Check(i, Equals, 2)
}

func (s *s) TestState_GetUsers(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	st.addUser(users[0])
	st.addUser(users[1])
	c.Check(len(st.GetUsers()), Equals, 2)
	for _, u := range st.GetUsers() {
		has := false
		for _, user := range users {
			if user == u {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
	}
}

func (s *s) TestState_GetChannels(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	c.Check(len(st.GetChannels()), Equals, 2)
	for _, ch := range st.GetChannels() {
		has := false
		for _, channel := range channels {
			if channel == ch {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
	}
}

func (s *s) TestState_GetUserChans(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetUserChans(users[0]), IsNil)
	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	c.Check(len(st.GetUserChans(users[0])), Equals, 2)
	for _, uc := range st.GetUserChans(users[0]) {
		has := false
		for _, channel := range channels {
			if channel == uc {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
	}
}

func (s *s) TestState_GetChanUsers(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.GetChanUsers(channels[0]), IsNil)
	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	c.Check(len(st.GetChanUsers(channels[0])), Equals, 2)
	for _, cu := range st.GetChanUsers(channels[0]) {
		has := false
		for _, user := range users {
			if user == cu {
				has = true
				break
			}
		}
		c.Check(has, Equals, true)
	}
}

func (s *s) TestState_IsOn(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.addChannel(channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.addUser(users[0])
	st.addToChannel(users[0], channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
}

func (s *s) TestState_UpdateNick(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.NICK,
		Sender: users[0],
		Args:   []string{nicks[1]},
	}

	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUser(users[1]), IsNil)
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.IsOn(users[1], channels[0]), Equals, false)
	for nick := range st.channelUsers[strings.ToLower(channels[0])] {
		c.Check(nick, Equals, nicks[0])
	}

	st.Update(ev)

	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.GetUser(users[1]), NotNil)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)
	for nick := range st.channelUsers[strings.ToLower(channels[0])] {
		c.Check(nick, Equals, nicks[1])
	}

	ev.Sender = users[0]
	ev.Args = []string{"newnick"}
	st.Update(ev)
	c.Check(st.GetUser("newnick"), NotNil)
	c.Check(st.GetUser(nicks[0]), IsNil)
}

func (s *s) TestState_UpdateNickSelfNilMaps(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.NICK,
		Sender: users[0],
		Args:   []string{nicks[1]},
	}
	st.addUser(users[0])
	st.Update(ev)

	_, ok := st.userChannels[nicks[0]]
	c.Check(ok, Equals, false)
	_, ok = st.userChannels[nicks[1]]
	c.Check(ok, Equals, false)
}

func (s *s) TestState_UpdateJoin(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.JOIN,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	st.addChannel(channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.Update(ev)
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)

	st, _ = NewState(netInfo)
	st.Self = self
	st.addChannel(channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.Update(ev)
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
}

func (s *s) TestState_UpdateJoinSelf(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.JOIN,
		Sender: self.Host(),
		Args:   []string{channels[0]},
	}

	c.Check(st.GetChannel(channels[0]), IsNil)
	c.Check(st.IsOn(st.Self.Nick(), channels[0]), Equals, false)
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]), NotNil)
	c.Check(st.IsOn(st.Self.Nick(), channels[0]), Equals, true)
}

func (s *s) TestState_UpdatePart(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.PART,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	st.addUser(users[0])
	st.addUser(users[1])

	// Test coverage, make sure adding to a channel that doesn't exist does
	// nothing.
	st.addToChannel(users[0], channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)

	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	st.addToChannel(users[0], channels[1])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)
	c.Check(st.IsOn(users[0], channels[1]), Equals, true)
	c.Check(st.IsOn(users[1], channels[1]), Equals, false)

	st.Update(ev)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)
	c.Check(st.IsOn(users[0], channels[1]), Equals, true)
	c.Check(st.IsOn(users[1], channels[1]), Equals, false)

	ev.Sender = users[1]
	st.Update(ev)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[1], channels[0]), Equals, false)
	c.Check(st.IsOn(users[0], channels[1]), Equals, true)
	c.Check(st.IsOn(users[1], channels[1]), Equals, false)

	ev.Sender = users[0]
	ev.Args[0] = channels[1]
	st.Update(ev)

	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[1], channels[0]), Equals, false)
	c.Check(st.IsOn(users[0], channels[1]), Equals, false)
	c.Check(st.IsOn(users[1], channels[1]), Equals, false)
}

func (s *s) TestState_UpdatePartSelf(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.PART,
		Sender: self.Host(),
		Args:   []string{channels[0]},
	}

	st.addUser(users[0])
	st.addUser(self.Host())
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	st.addToChannel(self.Nick(), channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.IsOn(users[0], channels[1]), Equals, true)
	c.Check(st.IsOn(self.Nick(), channels[0]), Equals, true)
	st.Update(ev)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[0], channels[1]), Equals, true)
	c.Check(st.IsOn(self.Nick(), channels[0]), Equals, false)
}

func (s *s) TestState_UpdateQuit(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.QUIT,
		Sender: users[0],
		Args:   []string{"quit message"},
	}

	// Test Quitting when we don't know the user
	st.Update(ev)
	c.Check(st.GetUser(users[0]), IsNil)

	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)
	c.Check(st.GetUser(users[1]), NotNil)

	st.Update(ev)

	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)
	c.Check(st.GetUser(users[1]), NotNil)

	ev.Sender = users[1]
	st.Update(ev)

	c.Check(st.IsOn(users[1], channels[0]), Equals, false)
	c.Check(st.GetUser(users[1]), IsNil)
}

func (s *s) TestState_UpdateKick(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.KICK,
		Sender: users[1],
		Args:   []string{channels[0], users[0]},
	}

	st.addUser(users[0])
	st.addUser(users[1])

	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	st.Update(ev)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
}

func (s *s) TestState_UpdateKickSelf(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.KICK,
		Sender: users[1],
		Args:   []string{channels[0], st.Self.Nick()},
	}

	st.addUser(st.Self.Host())
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	c.Check(st.GetChannel(channels[0]), NotNil)
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]), IsNil)
}

func (s *s) TestState_UpdateMode(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.MODE,
		Sender: users[0],
		Args: []string{channels[0],
			"+ovmb-vn", nicks[0], nicks[0], "*!*mask", nicks[1],
		},
		NetworkInfo: netInfo,
	}

	fail := st.GetUsersChannelModes(users[0], channels[0])
	c.Check(fail, IsNil)

	st.addChannel(channels[0])
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])

	u1modes := st.GetUsersChannelModes(users[0], channels[0])
	u2modes := st.GetUsersChannelModes(users[1], channels[0])
	u2modes.SetMode('v')
	st.GetChannel(channels[0]).Set("n")

	c.Check(st.GetChannel(channels[0]).IsSet("n"), Equals, true)
	c.Check(st.GetChannel(channels[0]).IsSet("mb"), Equals, false)
	c.Check(u1modes.HasMode('o'), Equals, false)
	c.Check(u1modes.HasMode('v'), Equals, false)
	c.Check(u2modes.HasMode('v'), Equals, true)
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]).IsSet("n"), Equals, false)
	c.Check(st.GetChannel(channels[0]).IsSet("mb *!*mask"), Equals, true)
	c.Check(u1modes.HasMode('o'), Equals, true)
	c.Check(u1modes.HasMode('v'), Equals, true)
	c.Check(u2modes.HasMode('v'), Equals, false)
}

func (s *s) TestState_UpdateModeSelf(c *C) {
	st, err := NewState(netInfo)
	st.Self.User = self.User
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:        irc.MODE,
		Sender:      self.Host(),
		Args:        []string{self.Nick(), "+i-o"},
		NetworkInfo: netInfo,
	}

	st.Self.Set("o")

	c.Check(st.Self.IsSet("i"), Equals, false)
	c.Check(st.Self.IsSet("o"), Equals, true)
	st.Update(ev)
	c.Check(st.Self.IsSet("i"), Equals, true)
	c.Check(st.Self.IsSet("o"), Equals, false)
}

func (s *s) TestState_UpdateTopic(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.TOPIC,
		Sender: users[1],
		Args:   []string{channels[0], "topic topic"},
	}

	st.addChannel(channels[0])

	c.Check(st.GetChannel(channels[0]).Topic(), Equals, "")
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]).Topic(), Equals, "topic topic")
}

func (s *s) TestState_UpdateRplTopic(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.RPL_TOPIC,
		Sender: network,
		Args:   []string{self.Nick(), channels[0], "topic topic"},
	}

	st.addChannel(channels[0])

	c.Check(st.GetChannel(channels[0]).Topic(), Equals, "")
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]).Topic(), Equals, "topic topic")
}

func (s *s) TestState_UpdateEmptyTopic(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.TOPIC,
		Sender: users[1],
		Args:   []string{channels[0], ""},
	}

	ch := st.addChannel(channels[0])
	ch.SetTopic("topic topic")

	c.Check(st.GetChannel(channels[0]).Topic(), Equals, "topic topic")
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]).Topic(), Equals, "")
}

func (s *s) TestState_UpdatePrivmsg(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:        irc.PRIVMSG,
		Sender:      users[0],
		Args:        []string{channels[0]},
		NetworkInfo: netInfo,
	}

	st.addChannel(channels[0])

	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), IsNil)
	st.Update(ev)
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), NotNil)

	ev.Sender = network
	size := len(st.users)
	st.Update(ev)
	c.Check(len(st.users), Equals, size)
}

func (s *s) TestState_UpdateNotice(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:        irc.NOTICE,
		Sender:      users[0],
		Args:        []string{channels[0]},
		NetworkInfo: netInfo,
	}

	st.addChannel(channels[0])

	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), IsNil)
	st.Update(ev)
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), NotNil)

	ev.Sender = network
	size := len(st.users)
	st.Update(ev)
	c.Check(len(st.users), Equals, size)
}

func (s *s) TestState_UpdateWelcome(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)
	ev := &irc.Event{
		Name:   irc.RPL_WELCOME,
		Sender: network,
		Args:   []string{nicks[1], "Welcome to"},
	}

	st.Update(ev)
	c.Check(st.Self.Host(), Equals, nicks[1])
	c.Check(st.users[nicks[1]].Host(), Equals, st.Self.Host())

	ev = &irc.Event{
		Name:   irc.RPL_WELCOME,
		Sender: network,
		Args:   []string{nicks[1], "Welcome to " + users[1]},
	}

	st.Update(ev)
	c.Check(st.Self.Host(), Equals, users[1])
	c.Check(st.users[nicks[1]].Host(), Equals, st.Self.Host())
}

func (s *s) TestState_UpdateRplNamereply(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.RPL_NAMREPLY,
		Sender: network,
		Args: []string{
			self.Nick(), "=", channels[0],
			"@" + nicks[0] + " +" + nicks[1] + " " + self.Nick(),
		},
	}

	st.addChannel(channels[0])

	c.Check(st.GetUsersChannelModes(users[0], channels[0]), IsNil)
	c.Check(st.GetUsersChannelModes(users[1], channels[0]), IsNil)
	c.Check(st.GetUsersChannelModes(self.Nick(), channels[0]), IsNil)
	st.Update(ev)
	c.Check(
		st.GetUsersChannelModes(users[0], channels[0]).String(), Equals, "o")
	c.Check(
		st.GetUsersChannelModes(users[1], channels[0]).String(), Equals, "v")
	c.Check(st.GetUsersChannelModes(
		self.Nick(), channels[0]).String(), Equals, "")
}

func (s *s) TestState_RplWhoReply(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.RPL_WHOREPLY,
		Sender: network,
		Args: []string{
			self.Nick(), channels[0], irc.Username(users[0]),
			irc.Hostname(users[0]), "*.network.net", nicks[0], "Hx@d",
			"3 real name",
		},
	}

	st.addChannel(channels[0])

	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.GetUsersChannelModes(users[0], channels[0]), IsNil)
	st.Update(ev)
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUser(users[0]).Host(), Equals, users[0])
	c.Check(st.GetUser(users[0]).Realname(), Equals, "real name")
	c.Check(
		st.GetUsersChannelModes(users[0], channels[0]).String(), Equals, "o")
}

func (s *s) TestState_UpdateRplMode(c *C) {
	st, err := NewState(netInfo)
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.RPL_CHANNELMODEIS,
		Sender: network,
		Args:   []string{self.Nick(), channels[0], "+ntzl", "10"},
	}

	st.addChannel(channels[0])
	c.Check(st.GetChannel(channels[0]).IsSet("ntzl 10"), Equals, false)
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]).IsSet("ntzl 10"), Equals, true)
}

func (s *s) TestState_UpdateRplBanlist(c *C) {
	st, err := NewState(netInfo)
	st.Self = self
	c.Check(err, IsNil)

	ev := &irc.Event{
		Name:   irc.RPL_BANLIST,
		Sender: network,
		Args: []string{self.Nick(), channels[0], nicks[0] + "!*@*", nicks[1],
			"1367197165"},
	}

	st.addChannel(channels[0])
	c.Check(st.GetChannel(channels[0]).HasBan(nicks[0]+"!*@*"), Equals, false)
	st.Update(ev)
	c.Check(st.GetChannel(channels[0]).HasBan(nicks[0]+"!*@*"), Equals, true)
}
