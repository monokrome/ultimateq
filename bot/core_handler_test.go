package bot

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

//===================================================================
// Fixtures for basic responses as well as full bot required messages
//===================================================================
type testPoint struct {
	irc.Helper
	buf *bytes.Buffer
	srv *Server
}

func makeTestPoint(srv *Server) *testPoint {
	buf := &bytes.Buffer{}
	t := &testPoint{irc.Helper{buf}, buf, srv}
	return t
}

func (t *testPoint) gets() string {
	return t.buf.String()
}

func (t *testPoint) resetTestWritten() {
	t.buf.Reset()
}

func (t *testPoint) GetKey() string {
	return netID
}

//==============
// Tests
//==============
func TestCoreHandler_Ping(t *testing.T) {
	handler := coreHandler{}
	ev := irc.NewEvent(netID, netInfo, irc.PING, "", "123123123123")
	endpoint := makeTestPoint(nil)
	handler.HandleRaw(endpoint, ev)
	expect := irc.PONG + " :" + ev.Args[0]
	if got := endpoint.gets(); got != expect {
		t.Errorf("Expected: %s, got: %s", expect, got)
	}
}

func TestCoreHandler_Connect(t *testing.T) {
	cnf := fakeConfig.Clone()
	net := cnf.Network(netID).SetPassword("password")

	ch1 := config.Channel{Name: "#channel1", Password: "pass"}
	ch2 := config.Channel{Name: "#channel2"}
	net.SetChannels([]config.Channel{ch1, ch2})

	b, _ := createBot(cnf, nil, nil, devNull, false, false)

	password, _ := net.Password()
	nick, _ := net.Nick()
	username, _ := net.Username()
	realname, _ := net.Realname()

	handler := coreHandler{bot: b}
	msg1 := fmt.Sprintf("PASS :%v", password)
	msg2 := fmt.Sprintf("NICK :%v", nick)
	msg3 := fmt.Sprintf("USER %v 0 * :%v", username, realname)
	msg4 := fmt.Sprintf("JOIN %v %v", ch1.Name, ch1.Password)
	msg5 := fmt.Sprintf("JOIN %v", ch2.Name)

	ev := irc.NewEvent(netID, netInfo, irc.CONNECT, "")
	endpoint := makeTestPoint(b.servers[netID])
	handler.HandleRaw(endpoint, ev)

	expect := msg1 + msg2 + msg3 + msg4 + msg5
	if got := endpoint.gets(); got != expect {
		t.Errorf("Expected: %s, got: %s", expect, got)
	}

	endpoint.resetTestWritten()

	net.SetNoAutoJoin(true)
	handler.HandleRaw(endpoint, ev)
	expect = msg1 + msg2 + msg3
	if got := endpoint.gets(); got != expect {
		t.Errorf("Expected: %s, got: %s", expect, got)
	}
}

func TestCoreHandler_Nick(t *testing.T) {
	b, _ := createBot(fakeConfig, nil, nil, devNull, false, false)
	cnf := fakeConfig.Network(netID)
	handler := coreHandler{bot: b}
	ev := irc.NewEvent(netID, netInfo, irc.ERR_NICKNAMEINUSE, "")

	endpoint := makeTestPoint(b.servers[netID])

	nick, _ := cnf.Nick()
	altnick, _ := cnf.Altnick()
	nickstr := "NICK :"
	nick1 := nickstr + altnick
	nick2 := nickstr + nick + "_"
	nick3 := nickstr + nick + "__"

	handler.HandleRaw(endpoint, ev)
	if got := endpoint.gets(); got != nick1 {
		t.Errorf("Expected: %s, got: %s", nick1, got)
	}
	endpoint.resetTestWritten()
	handler.HandleRaw(endpoint, ev)
	if got := endpoint.gets(); got != nick2 {
		t.Errorf("Expected: %s, got: %s", nick2, got)
	}
	endpoint.resetTestWritten()
	handler.HandleRaw(endpoint, ev)
	if got := endpoint.gets(); got != nick3 {
		t.Errorf("Expected: %s, got: %s", nick3, got)
	}
}

func TestCoreHandler_Rejoin(t *testing.T) {
	cnf := fakeConfig.Clone()
	net := cnf.Network(netID).SetPassword("password").SetNoState(false).
		SetNoAutoJoin(true)

	nick, _ := net.Nick()
	ch1 := config.Channel{Name: "#channel1", Password: "pass"}
	ch2 := config.Channel{Name: "#channel2"}

	b, _ := createBot(cnf, nil, nil, devNull, false, false)
	st := b.servers[netID].state
	st.Update(
		irc.NewEvent(netID, netInfo, irc.RPL_WELCOME, "", "stuff", nick+"!a@b"),
	)

	endpoint := makeTestPoint(b.servers[netID])
	banned := irc.NewEvent(netID, netInfo, irc.ERR_BANNEDFROMCHAN, netID,
		nick, ch1.Name, "Banned message")
	kicked := irc.NewEvent(netID, netInfo, irc.KICK, "badguy",
		ch2.Name, nick, "Kick Message")

	handler := coreHandler{bot: b}
	handler.HandleRaw(endpoint, banned)
	handler.HandleRaw(endpoint, kicked)

	if got := endpoint.gets(); len(got) > 0 {
		t.Error("Expected nothing to happen with noautojoin set.")
	}

	handler.HandleRaw(endpoint, banned)
	handler.HandleRaw(endpoint, kicked)

	net.SetNoAutoJoin(false)

	if got := endpoint.gets(); len(got) > 0 {
		t.Error("Expected nothing to happen without channels set.")
	}

	net.SetChannels([]config.Channel{ch1, ch2})

	handler.HandleRaw(endpoint, banned)
	handler.HandleRaw(endpoint, kicked)

	exp1 := fmt.Sprintf("JOIN %v %v", ch1.Name, ch1.Password)
	exp2 := fmt.Sprintf("JOIN %v", ch2.Name)
	got := endpoint.gets()
	if !strings.Contains(got, exp1) {
		t.Error("Expected it to have joined #channel1 after ban.")
	}
	if !strings.Contains(got, exp2) {
		t.Error("Expected it to have joined #channel1 after kick.")
	}
}

func TestCoreHandler_NetInfo(t *testing.T) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, _ := createBot(fakeConfig, connProvider, nil, devNull, true, false)

	msg1 := irc.NewEvent(netID, netInfo, irc.RPL_MYINFO, "",
		"NICK", "irc.test.net", "testircd-1.2", "acCior", "beiIklmno")
	msg2 := irc.NewEvent(netID, netInfo, irc.RPL_ISUPPORT, "",
		"RFC8213", "CHANTYPES=&$")
	srv := b.servers[netID]
	srv.handler.HandleRaw(&testPoint{}, msg1)
	srv.handler.HandleRaw(&testPoint{}, msg2)
	if got, exp := srv.netInfo.ServerName(), "irc.test.net"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.IrcdVersion(), "testircd-1.2"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.Usermodes(), "acCior"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.LegacyChanmodes(), "beiIklmno"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.Chantypes(), "&$"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
}

func TestCoreHandler_Join(t *testing.T) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, _ := createBot(fakeConfig, connProvider, nil, devNull, true, false)
	srv := b.servers[netID]

	srv.state.Self.User = data.NewUser("nick!user@host")
	ev := irc.NewEvent(netID, netInfo, irc.JOIN,
		srv.state.Self.Host(), "#chan")

	endpoint := makeTestPoint(nil)
	srv.handler.HandleRaw(endpoint, ev)
	if got, exp := endpoint.gets(), "WHO :#chanMODE :#chan"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
}
