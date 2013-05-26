package bot

import (
	"code.google.com/p/gomock/gomock"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	mocks "github.com/aarondl/ultimateq/inet/test"
	"github.com/aarondl/ultimateq/irc"
	"io"
	. "launchpad.net/gocheck"
	"net"
)

//===================================================================
// Fixtures for basic responses as well as full bot required messages
//===================================================================
var testWritten []string = make([]string, 0, 10)

type testSender struct {
}

func resetTestWritten() {
	testWritten = testWritten[:0]
}

func (t testSender) GetKey() string {
	return serverId
}

func (t testSender) Writeln(str string) error {
	testWritten = append(testWritten, str)
	return nil
}

// callBack gets the Bot, the mock Connection, the server's config, and the
// write channel if during the call to testHandlerResponse startWriter was true
type callBack func(*Bot, *mocks.MockConn, *config.Server, chan []byte)

/* WARNING:
 This test requires that we be able to wait on the socket to receive some data.
 Because of that, the mock must be modified.

 The two following places should have code injected:

 type MockConn struct {
	 ...
	 Writechan chan []byte
 }

 func (_m *MockConn) Write(_param0 []byte) (int, error) {
	 ret := _m.ctrl.Call(_m, "Write", _param0)
	 if _m.Writechan != nil {
		 _m.Writechan <- _param0
	 }
	 ...
 }
*/
func testHandlerResponse(c *C, startWriter, startReader bool,
	before callBack, after callBack) {

	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	var channel chan []byte
	if startWriter {
		conn.Writechan = make(chan []byte)
		channel = conn.Writechan
	}

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)

	server := b.servers[serverId]

	if before != nil {
		before(b, conn, server.conf, channel)
	}

	conn.EXPECT().Close()

	b.Connect()
	b.start(startWriter, startReader)

	if after != nil {
		after(b, conn, server.conf, channel)
	}

	b.WaitForHalt()
	if startReader {
		b.Stop()
	}
	b.Disconnect()
}

//==============
// Tests
//==============
func (s *s) TestCoreHandler_Ping(c *C) {
	handler := coreHandler{}
	msg := &irc.IrcMessage{
		Name: "PING",
		Args: []string{"123123123123"},
	}
	handler.HandleRaw(msg, testSender{})
	c.Assert(testWritten[0], Equals, "PONG :"+msg.Args[0])
}

func (s *s) TestCoreHandler_Connect(c *C) {
	testHandlerResponse(c, true, false,
		func(_ *Bot, conn *mocks.MockConn, conf *config.Server, c chan []byte) {
			msg1 := fmt.Sprintf("NICK :%v\r\n", conf.GetNick())
			msg2 := fmt.Sprintf("USER %v 0 * :%v\r\n",
				conf.GetUsername(), conf.GetRealname())

			gomock.InOrder(
				conn.EXPECT().Write([]byte(msg1)).Return(len(msg1), nil),
				conn.EXPECT().Write([]byte(msg2)).Return(len(msg2), io.EOF),
			)
		},
		func(_ *Bot, conn *mocks.MockConn, conf *config.Server, c chan []byte) {
			<-c
			<-c
		},
	)
}

func (s *s) TestCoreHandler_Nick(c *C) {
	testHandlerResponse(c, true, true,
		func(_ *Bot, conn *mocks.MockConn, conf *config.Server, c chan []byte) {
			nickstr := "NICK :%v\r\n"
			nick1 := fmt.Sprintf(nickstr, conf.GetNick())
			nick2 := fmt.Sprintf(nickstr, conf.GetAltnick())
			nick3 := fmt.Sprintf(nickstr, conf.GetNick()+"_")
			nick4 := fmt.Sprintf(nickstr, conf.GetNick()+"__")
			user := fmt.Sprintf("USER %v 0 * :%v\r\n",
				conf.GetUsername(), conf.GetRealname())
			errmsg := fmt.Sprintf("433 :Nick is in use\r\n")

			mocks.ByteFiller = []byte(
				errmsg + errmsg + errmsg + errmsg,
			)

			conn.EXPECT().Write([]byte(nick1)).Return(len(nick1), nil)
			conn.EXPECT().Write([]byte(user)).Return(len(user), nil)
			conn.EXPECT().Write([]byte(nick2)).Return(len(nick2), nil)
			conn.EXPECT().Write([]byte(nick3)).Return(len(nick3), nil)
			conn.EXPECT().Write([]byte(nick4)).Return(len(nick4), io.EOF)
			gomock.InOrder(
				conn.EXPECT().Read(gomock.Any()).Return(len(errmsg), nil),
				conn.EXPECT().Read(gomock.Any()).Return(len(errmsg), nil),
				conn.EXPECT().Read(gomock.Any()).Return(len(errmsg), io.EOF),
			)
		},
		func(_ *Bot, conn *mocks.MockConn, conf *config.Server, c chan []byte) {
			for i := 0; i < 5; i++ {
				<-c
			}
		},
	)
}

func (s *s) TestCoreHandler_005(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)

	msg := &irc.IrcMessage{
		Name: "005",
		Args: []string{"RFC8213", "CHANTYPES=&$"},
	}
	srv := b.servers[serverId]
	srv.handler.HandleRaw(msg, testSender{})
	c.Assert(srv.caps.Chantypes(), Equals, "&$")
}