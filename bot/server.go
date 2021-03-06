package bot

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"github.com/inconshreveable/log15"
)

// Status is the status of a network connection.
type Status byte

// Server Statuses
const (
	STATUS_STOPPED Status = iota
	STATUS_CONNECTING
	STATUS_STARTED
	STATUS_RECONNECTING
)

const (
	// errServerAlreadyConnected occurs if a server has not been shutdown
	// before another attempt to connect to it is made.
	errFmtAlreadyConnected = "bot: %v already connected.\n"
)

var (
	// errNotConnected happens when a write occurs to a disconnected server.
	errNotConnected = errors.New("bot: Server not connected")
	// errFailedToLoadCertificate happens when we fail to parse the certificate
	errFailedToLoadCertificate = errors.New("bot: Failed to load certificate")
	// errServerKilledConn happens when the server is killed mid-connect.
	errServerKilledConn = errors.New("bot: Killed trying to connect")
)

// connResult is used to return results from the channel patterns in
// createIrcClient
type connResult struct {
	conn      net.Conn
	temporary bool
	err       error
}

// certReader is for IoC of the createTlsConfig function.
type certReader func(string) (*x509.CertPool, error)

// Server is all the details around a specific server connection. Also contains
// the connection and configuration for the specific server.
type Server struct {
	bot       *Bot
	networkID string

	log15.Logger

	// Status
	status          Status
	statusListeners [][]chan Status

	// Configuration
	conf    *config.Config
	netInfo *irc.NetworkInfo

	// Dispatching
	dispatchCore *dispatch.DispatchCore
	dispatcher   *dispatch.Dispatcher
	cmds         *cmd.Cmds
	writer       irc.Writer

	handlerID int
	handler   *coreHandler

	// State and Connection
	client      *inet.IrcClient
	state       *data.State
	started     bool
	serverIndex int
	reconnScale time.Duration
	killable    chan int

	// protects client reading/writing
	protect sync.RWMutex

	// protects the state from reading and writing.
	protectState sync.RWMutex
}

// Write writes to the server's IrcClient.
func (s *Server) Write(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	s.protect.RLock()
	defer s.protect.RUnlock()

	if s.GetStatus() != STATUS_STOPPED {
		return s.client.Write(buf)
	}

	return 0, errNotConnected
}

// createDispatcher uses the server's current ProtoCaps to create a dispatcher.
func (s *Server) createDispatching(prefix rune, channels []string) {
	s.dispatchCore = dispatch.NewDispatchCore(s.Logger, channels...)
	s.dispatcher = dispatch.NewDispatcher(s.dispatchCore)
	s.cmds = cmd.NewCmds(prefix, s.dispatchCore)
}

// createState uses the server's current ProtoCaps to create a state.
func (s *Server) createState() (err error) {
	s.state, err = data.NewState(s.netInfo)
	return err
}

// createIrcClient connects to the configured server, and creates an IrcClient
// for use with that connection.
func (s *Server) createIrcClient() (error, bool) {
	if s.client != nil {
		return fmt.Errorf(errFmtAlreadyConnected, s.networkID), false
	}

	var result *connResult
	resultService := make(chan chan *connResult)
	resultChan := make(chan *connResult)

	go s.createConnection(resultService)

	select {
	case resultService <- resultChan:
		result = <-resultChan
		if result.err != nil {
			return result.err, result.temporary
		}
	case s.killable <- 0:
		close(resultService)
		return errServerKilledConn, false
	}

	cfg := s.conf.Network(s.networkID)
	floodPenalty, _ := cfg.FloodLenPenalty()
	floodTimeout, _ := cfg.FloodTimeout()
	floodStep, _ := cfg.FloodStep()
	keepAlive, _ := cfg.KeepAlive()

	s.protect.Lock()
	s.client = inet.NewIrcClient(
		result.conn,
		s.Logger,
		int(floodPenalty),
		time.Duration(floodTimeout)*time.Second,
		time.Duration(floodStep)*time.Second,
		time.Duration(keepAlive)*time.Second,
		time.Second,
	)
	s.protect.Unlock()
	return nil, false
}

// createConnection creates a connection based off the server receiver's
// config variables. It takes a chan of channels to return the result on.
// If the channel is closed before it can send it's result, it will close the
// connection automatically.
func (s *Server) createConnection(resultService chan chan *connResult) {
	r := &connResult{}

	cfg := s.conf.Network(s.networkID)
	srvs, _ := cfg.Servers()
	ssl, _ := cfg.SSL()
	if s.serverIndex >= len(srvs) {
		s.serverIndex = 0
	}

	server := srvs[s.serverIndex]
	s.Info("Connecting", "host", server)

	if s.bot.connProvider == nil {
		if ssl {
			var conf *tls.Config
			conf, r.err = s.createTlsConfig(readCert)
			if r.err == nil {
				r.conn, r.err = tls.Dial("tcp", server, conf)
			}
		} else {
			r.conn, r.err = net.Dial("tcp", server)
		}
	} else {
		r.conn, r.err = s.bot.connProvider(server)
	}

	if r.err != nil {
		s.Error("Failed to connect", "host", server)
		if e, ok := r.err.(net.Error); ok {
			r.temporary = e.Temporary()
		} else {
			r.temporary = false
		}
		s.serverIndex++
	}

	if resultChan, ok := <-resultService; ok {
		resultChan <- r
	} else {
		if r.conn != nil {
			r.conn.Close()
		}
	}
}

// createTlsConfig creates a tls config appropriate for the
func (s *Server) createTlsConfig(cr certReader) (conf *tls.Config, err error) {
	conf = &tls.Config{}
	cfg := s.conf.Network(s.networkID)
	skipVerify, _ := cfg.NoVerifyCert()
	conf.InsecureSkipVerify = skipVerify

	if cert, ok := cfg.SSLCert(); ok {
		conf.RootCAs, err = cr(cert)
	}

	return
}

// Close shuts down the connection and returns.
func (s *Server) Close() (err error) {
	s.protect.Lock()
	defer s.protect.Unlock()

	if s.client != nil {
		err = s.client.Close()
	}
	s.client = nil
	return
}

// rehashNetworkInfo delivers updated information to the server's components who
// may need it.
func (s *Server) rehashNetworkInfo() error {
	var err error
	s.protectState.Lock()
	defer s.protectState.Unlock()
	if s.state != nil {
		err = s.state.SetNetworkInfo(s.netInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

// setStatus safely sets the status of the server and notifies any listeners.
func (s *Server) setStatus(newstatus Status) {
	s.protect.Lock()
	defer s.protect.Unlock()

	s.status = newstatus
	if s.statusListeners == nil {
		return
	}
	for _, listener := range s.statusListeners[0] {
		listener <- s.status
	}
	i := byte(newstatus) + 1
	for _, listener := range s.statusListeners[i] {
		listener <- s.status
	}
}

// addStatusListener adds a listener for status changes.
func (s *Server) addStatusListener(listener chan Status, listen ...Status) {
	s.protect.Lock()
	defer s.protect.Unlock()

	if s.statusListeners == nil {
		s.statusListeners = [][]chan Status{
			make([]chan Status, 0),
			make([]chan Status, 0),
			make([]chan Status, 0),
			make([]chan Status, 0),
			make([]chan Status, 0),
		}
	}

	if len(listen) == 0 {
		s.statusListeners[0] = append(s.statusListeners[0], listener)
	} else {
		for _, st := range listen {
			i := byte(st) + 1
			s.statusListeners[i] = append(s.statusListeners[i], listener)
		}
	}
}

// GetStatus safely gets the status of the server.
func (s *Server) GetStatus() Status {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return s.status
}

// readCert returns a CertPool containing the client certificate specified
// in filename.
func readCert(filename string) (certpool *x509.CertPool, err error) {
	var pem []byte
	var file *os.File

	if file, err = os.Open(filename); err != nil {
		return
	}

	defer file.Close()

	pem, err = ioutil.ReadAll(file)
	if err != nil {
		return
	}

	certpool = x509.NewCertPool()
	ok := certpool.AppendCertsFromPEM(pem)
	if !ok {
		err = errFailedToLoadCertificate
	}
	return
}
