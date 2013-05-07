package bot

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

func (s *s) TestConfig(c *C) {
	config := CreateConfig()
	c.Assert(config.Servers, NotNil)
	c.Assert(config.defaults, NotNil)
}

func (s *s) TestConfig_Fallbacks(c *C) {
	config := CreateConfig()
	parent, host, ssl, verifyCert := config, "irc.com", true, true
	var port uint = 10
	nick, altnick, realname, userhost, prefix := "a", "b", "c", "d", "e"
	chans := []string{"#chan1", "#chan2"}

	config.defaults = &irc{
		port, ssl, true, verifyCert, true,
		nick, altnick, realname, userhost, prefix, chans,
	}

	irc := &irc{}
	c.Assert(irc.port, Equals, uint(0))
	c.Assert(irc.ssl, Equals, false)
	c.Assert(irc.isSslSet, Equals, false)
	c.Assert(irc.verifyCert, Equals, false)
	c.Assert(irc.isVerifyCertSet, Equals, false)
	c.Assert(irc.nick, Equals, "")
	c.Assert(irc.altnick, Equals, "")
	c.Assert(irc.realname, Equals, "")
	c.Assert(irc.userhost, Equals, "")
	c.Assert(irc.prefix, Equals, "")
	c.Assert(irc.channels, IsNil)

	server := &Server{parent, host, irc}
	config.Servers[host] = server

	c.Assert(server.GetHost(), Equals, host)
	c.Assert(server.GetPort(), Equals, port)
	c.Assert(server.GetSsl(), Equals, config.defaults.ssl)
	c.Assert(server.GetVerifyCert(), Equals, config.defaults.verifyCert)
	c.Assert(server.GetNick(), Equals, config.defaults.nick)
	c.Assert(server.GetAltnick(), Equals, config.defaults.altnick)
	c.Assert(server.GetRealname(), Equals, config.defaults.realname)
	c.Assert(server.GetUserhost(), Equals, config.defaults.userhost)
	c.Assert(server.GetPrefix(), Equals, config.defaults.prefix)
	c.Assert(len(server.GetChannels()), Equals, len(config.defaults.channels))
	for i, v := range server.GetChannels() {
		c.Assert(v, Equals, config.defaults.channels[i])
	}

	//Check default bools more throughly
	server.irc.isSslSet = true
	server.irc.isVerifyCertSet = true
	c.Assert(server.GetSsl(), Equals, false)
	c.Assert(server.GetVerifyCert(), Equals, false)

	server.irc.isSslSet = false
	server.irc.isVerifyCertSet = false
	config.defaults.ssl = false
	config.defaults.verifyCert = false
	c.Assert(server.GetSsl(), Equals, false)
	c.Assert(server.GetVerifyCert(), Equals, false)

	//Check default port more thoroughly
	config.defaults.port = 0
	c.Assert(server.GetPort(), Equals, uint(ircDefaultPort))
}

func (s *s) TestConfig_Fluent(c *C) {
	srv1 := Server{
		nil, "irc.gamesurge.net",
		&irc{
			5555, true, true, false, true, "n1", "a1", "r1", "h1", "p1",
			[]string{"#chan", "#chan2"},
		},
	}
	defs := Server{
		nil, "nuclearfallout.gamesurge.net",
		&irc{
			7777, false, false, true, false, "n2", "a2", "r2", "h2", "p2",
			[]string{"#chan2"},
		},
	}
	srv2 := "znc.gamesurge.net"

	conf := Configure().
		Port(defs.irc.port).
		Nick(defs.irc.nick).
		Altnick(defs.irc.altnick).
		Realname(defs.irc.realname).
		Userhost(defs.irc.userhost).
		Prefix(defs.irc.prefix).
		Channels(defs.irc.channels...).
		Server(srv1.host).
		Port(srv1.irc.port).
		Ssl(srv1.irc.ssl).
		VerifyCert(srv1.irc.verifyCert).
		Nick(srv1.irc.nick).
		Altnick(srv1.irc.altnick).
		Realname(srv1.irc.realname).
		Userhost(srv1.irc.userhost).
		Prefix(srv1.irc.prefix).
		Channels(srv1.irc.channels...).
		Server(srv2)

	server := conf.Servers[srv1.host]
	server2 := conf.Servers[srv2]
	c.Assert(server.GetHost(), Equals, srv1.GetHost())
	c.Assert(server.GetPort(), Equals, srv1.GetPort())
	c.Assert(server.GetSsl(), Equals, srv1.GetSsl())
	c.Assert(server.GetVerifyCert(), Equals, srv1.GetVerifyCert())
	c.Assert(server.GetNick(), Equals, srv1.GetNick())
	c.Assert(server.GetAltnick(), Equals, srv1.GetAltnick())
	c.Assert(server.GetRealname(), Equals, srv1.GetRealname())
	c.Assert(server.GetUserhost(), Equals, srv1.GetUserhost())
	c.Assert(server.GetPrefix(), Equals, srv1.GetPrefix())
	c.Assert(len(server.GetChannels()), Equals, len(srv1.GetChannels()))
	for i, v := range server.GetChannels() {
		c.Assert(v, Equals, srv1.irc.channels[i])
	}

	c.Assert(server2.GetHost(), Equals, srv2)
	c.Assert(server2.GetPort(), Equals, defs.GetPort())
	c.Assert(server2.GetSsl(), Equals, defs.GetSsl())
	c.Assert(server2.GetVerifyCert(), Equals, defs.GetVerifyCert())
	c.Assert(server2.GetNick(), Equals, defs.GetNick())
	c.Assert(server2.GetAltnick(), Equals, defs.GetAltnick())
	c.Assert(server2.GetRealname(), Equals, defs.GetRealname())
	c.Assert(server2.GetUserhost(), Equals, defs.GetUserhost())
	c.Assert(server2.GetPrefix(), Equals, defs.GetPrefix())
	c.Assert(len(server2.GetChannels()), Equals, len(defs.GetChannels()))
	for i, v := range server2.GetChannels() {
		c.Assert(v, Equals, defs.irc.channels[i])
	}
}

func (s *s) TestValidNames(c *C) {
	goodNicks := []string{`a1bc`, `a5bc`, `a9bc`, `MyNick`, `[MyNick`,
		`My[Nick`, `]MyNick`, `My]Nick`, `\MyNick`, `My\Nick`, "MyNick",
		"My`Nick", `_MyNick`, `My_Nick`, `^MyNick`, `My^Nick`, `{MyNick`,
		`My{Nick`, `|MyNick`, `My|Nick`, `}MyNick`, `My}Nick`,
	}

	badNicks := []string{`My Name`, `My!Nick`, `My"Nick`, `My#Nick`, `My$Nick`,
		`My%Nick`, `My&Nick`, `My'Nick`, `My/Nick`, `My(Nick`, `My)Nick`,
		`My*Nick`, `My+Nick`, `My,Nick`, `My-Nick`, `My.Nick`, `My/Nick`,
		`My;Nick`, `My:Nick`, `My<Nick`, `My=Nick`, `My>Nick`, `My?Nick`,
		`My@Nick`, `1abc`, `5abc`, `9abc`, `@ChanServ`,
	}

	for i := 0; i < len(goodNicks); i++ {
		if !nicknameRegex.MatchString(goodNicks[i]) {
			c.Errorf("Good nick failed regex: %v\n", goodNicks[i])
		}
	}
	for i := 0; i < len(badNicks); i++ {
		if nicknameRegex.MatchString(badNicks[i]) {
			c.Errorf("Bad nick passed regex: %v\n", badNicks[i])
		}
	}
}

func (s *s) TestValidChannels(c *C) {
	// Check that the first letter must be {#+!&}
	goodChannels := []string{"#ValidChannel", "+ValidChannel", "&ValidChannel",
		"!12345", "#c++"}

	badChannels := []string{"#Invalid Channel", "#Invalid,Channel",
		"#Invalid\aChannel", "#", "+", "&", "InvalidChannel"}

	for i := 0; i < len(goodChannels); i++ {
		if !channelRegex.MatchString(goodChannels[i]) {
			c.Errorf("Good chan failed regex: %v\n", goodChannels[i])
		}
	}
	for i := 0; i < len(badChannels); i++ {
		if channelRegex.MatchString(badChannels[i]) {
			c.Errorf("Bad chan passed regex: %v\n", badChannels[i])
		}
	}
}