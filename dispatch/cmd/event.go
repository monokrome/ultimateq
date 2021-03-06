package cmd

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

// Event represents the data about the event that occurred. The commander
// fills the Event structure with information about the user and channel
// involved. It also embeds the State and Store for easy access.
//
// Event comes with the implication that the State and Store
// have been locked for reading, A typical event handler that quickly does some
// work and returns does not need to worry about calling Close() since it is
// guaranteed to automatically be closed when the
// handler returns. But a call to Close() must be given in a
// command handler that will do some long running processes. Note that all data
// in the Event struct becomes volatile and not thread-safe after a call
// to Close() has been made, so the values in the Event struct are set to
// nil but extra caution should be made when copying data from this struct and
// calling Close() afterwards since this data is shared between other parts of
// the bot.
//
// Some parts of Event will be nil under certain circumstances so elements
// within must be checked for nil, see each element's documentation
// for further information.
type Event struct {
	*irc.Event
	locker data.Locker
	*data.State
	*data.Store
	// User can be nil if the bot's State is disabled.
	User *data.User
	// StoredUser will be nil when there is no required access.
	StoredUser *data.StoredUser
	// UserChannelModes will be nil when the message was not sent to a channel.
	UserChannelModes *data.UserModes
	// Channel will be nil when the message was not sent to a channel.
	Channel *data.Channel
	// TargetChannel will not be nil when the command has the #channel
	// parameter. The parameter can still be nil when the channel is not known
	// to the bot.
	TargetChannel *data.Channel
	// TargetUsers is populated when the arguments contain a ~nick argument, and
	// as a byproduct of looking up authentication, when the arguments contain
	// a *user argument, and a nickname is passed instead of a *username.
	TargetUsers map[string]*data.User
	// TargetStoredUser is populated when the arguments contain a *user
	// argument.
	TargetStoredUser map[string]*data.StoredUser
	// TargetVarUsers is populated when the arguments contain a ~nick...
	// argument. When a *user... parameter is used, it will be sparsely filled
	// whenever a user is requested by nickname not *username.
	TargetVarUsers []*data.User
	// TargetVarUsers is populated when the arguments contain a *user...
	// argument.
	TargetVarStoredUser []*data.StoredUser

	args map[string]string
	once sync.Once
}

// GetArg gets an argument that was passed in to the command by the user. The
// name of the argument passed into Register() is required to get the argument.
func (ev *Event) GetArg(arg string) string {
	return ev.args[arg]
}

// SplitArg behaves exactly like GetArg but calls strings.Fields on the
// argument. Useful for varargs...
func (ev *Event) SplitArg(arg string) (args []string) {
	if str, ok := ev.args[arg]; ok && len(str) > 0 {
		args = strings.Fields(str)
	}
	return
}

// FindUserByNick finds a user by their nickname. An error is returned if
// they were not found.
func (ev *Event) FindUserByNick(nick string) (*data.User, error) {
	if ev.State == nil {
		return nil, errors.New(errMsgStateDisabled)
	}

	user := ev.State.GetUser(nick)
	if user == nil {
		return nil, fmt.Errorf(errFmtUserNotFound, nick)
	}

	return user, nil
}

// FindAccessByUser locates a user's access based on their nick or
// username. To look up by username instead of nick use the * prefix before the
// username in the string. The user parameter is returned when a nickname lookup
// is done. An error occurs if the user is not found, the user is not authed,
// the username is not registered, etc.
func (ev *Event) FindAccessByUser(server, nickOrUser string) (
	access *data.StoredUser, user *data.User, err error) {
	if ev.Store == nil {
		err = errors.New(errMsgStoreDisabled)
		return
	}

	switch nickOrUser[0] {
	case '*':
		if len(nickOrUser) == 1 {
			err = errors.New(errMsgMissingUsername)
			return
		}
		uname := nickOrUser[1:]
		access, err = ev.Store.FindUser(uname)
		if access == nil {
			err = fmt.Errorf(errFmtUserNotRegistered, uname)
			return
		}
	default:
		if ev.State == nil {
			err = errors.New(errMsgStateDisabled)
			return
		}

		user = ev.State.GetUser(nickOrUser)
		if user == nil {
			err = fmt.Errorf(errFmtUserNotFound, nickOrUser)
			return
		}
		access = ev.Store.GetAuthedUser(server, user.Host())
		if access == nil {
			err = fmt.Errorf(errFmtUserNotAuthed, nickOrUser)
			return
		}
	}

	if err != nil {
		err = fmt.Errorf(errFmtInternal, err)
	}
	return
}

// Close closes the handles to the internal structures. Calling Close is not
// required. See Event's documentation for when to call this method.
// All Event's methods and fields become invalid after a call to Close.
// Close will never return an error so it can be safely ignored.
func (ev *Event) Close() error {
	ev.once.Do(func() {
		ev.User = nil
		ev.StoredUser = nil
		ev.UserChannelModes = nil
		ev.Channel = nil
		ev.State = nil
		ev.Store = nil
		ev.locker.CloseState(ev.NetworkID)
		ev.locker.CloseReadStore()
	})
	return nil
}
