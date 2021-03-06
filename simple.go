// The ultimateq bot framework.
package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/query"
	"github.com/aarondl/quotes"
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"github.com/inconshreveable/log15"
)

var (
	sanitizeNewline = strings.NewReplacer("\r\n", " ", "\n", " ")
	rgxSpace        = regexp.MustCompile(`\s{2,}`)
	queryConf       query.Config
)

const (
	dateFormat = "January 02, 2006 at 3:04pm MST"
)

/* =====================
 Helper methods.
===================== */
func sanitize(str string) string {
	return rgxSpace.ReplaceAllString(sanitizeNewline.Replace(str), " ")
}

type Quoter struct {
	db *quotes.QuoteDB
}

type Queryer struct {
}

type Runnable struct {
}

type Handler struct {
	b *bot.Bot
}

// Let reflection hook up the commands, instead of doing it here.
func (_ *Quoter) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

func (_ *Queryer) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

func (_ *Runnable) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

func (_ *Handler) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {
	return nil
}

/* =====================
 Quoter methods.
===================== */

func (q *Quoter) Addquote(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	quote := ev.GetArg("quote")
	if len(quote) == 0 {
		return nil
	}

	ev.Close()

	id, err := q.db.AddQuote(nick, quote)
	if err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else {
		w.Noticef(nick, "\x02Quote:\x02 Added quote #%d", id)
	}
	return nil
}

func (q *Quoter) Delquote(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	id, err := strconv.Atoi(ev.GetArg("id"))
	ev.Close()

	if err != nil {
		w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}
	if did, err := q.db.DelQuote(int(id)); err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else if !did {
		w.Notice(nick, "\x02Quote:\x02 Could not find quote %d.", id)
	} else {
		w.Noticef(nick, "\x02Quote:\x02 Quote %d deleted.", id)
	}
	return nil
}

func (q *Quoter) Editquote(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	quote := ev.GetArg("quote")
	id, err := strconv.Atoi(ev.GetArg("id"))
	ev.Close()

	if len(quote) == 0 {
		return nil
	}

	if err != nil {
		w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}
	if did, err := q.db.EditQuote(int(id), quote); err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else if !did {
		w.Notice(nick, "\x02Quote:\x02 Could not find quote %d.", id)
	} else {
		w.Noticef(nick, "\x02Quote:\x02 Quote %d updated.", id)
	}
	return nil
}

func (q *Quoter) Quote(w irc.Writer, ev *cmd.Event) error {
	strid := ev.GetArg("id")
	nick := ev.Nick()
	ev.Close()

	var quote string
	var id int
	var err error
	if len(strid) > 0 {
		getid, err := strconv.Atoi(strid)
		id = int(getid)
		if err != nil {
			w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
			return nil
		}
		quote, err = q.db.GetQuote(id)
	} else {
		id, quote, err = q.db.RandomQuote()
	}
	if err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
		return nil
	}

	if len(quote) == 0 {
		w.Notify(ev.Event, nick, "\x02Quote:\x02 Does not exist.")
	} else {
		w.Notifyf(ev.Event, nick, "\x02Quote (\x02#%d\x02):\x02 %s",
			id, quote)
	}
	return nil
}

func (q *Quoter) Quotes(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	ev.Close()

	w.Notifyf(ev.Event, nick, "\x02Quote:\x02 %d quote(s) in database.",
		q.db.NQuotes())
	return nil
}

func (q *Quoter) Details(w irc.Writer, ev *cmd.Event) error {
	nick := ev.Nick()
	id, err := strconv.Atoi(ev.GetArg("id"))
	ev.Close()

	if err != nil {
		w.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}

	if date, author, err := q.db.GetDetails(int(id)); err != nil {
		w.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else {
		w.Notifyf(ev.Event, nick,
			"\x02Quote (\x02#%d\x02):\x02 Created on %s by %s",
			id, time.Unix(date, 0).UTC().Format(dateFormat), author)
	}

	return nil
}

func (q *Quoter) Quoteweb(w irc.Writer, ev *cmd.Event) error {
	w.Notify(ev.Event, ev.Nick(), "\x02Quote:\x02 http://bitforge.ca:8000")
	return nil
}

/* =====================
 Queryer methods.
===================== */

func (_ *Queryer) PrivmsgChannel(w irc.Writer, ev *irc.Event) {
	if out, err := query.YouTube(ev.Message()); len(out) != 0 {
		w.Privmsg(ev.Target(), out)
	} else if err != nil {
		nick := ev.Nick()
		w.Notice(nick, err.Error())
	}
}

func (_ *Queryer) Calc(w irc.Writer, ev *cmd.Event) error {
	q := ev.GetArg("query")
	nick := ev.Nick()
	ev.Close()

	if out, err := query.Wolfram(q, &queryConf); len(out) != 0 {
		out = sanitize(out)

		// Ensure two lines only
		// ircmaxlen - maxhostsize - PRIVMSG - targetsize - spacing - colons
		maxlen := 2 * (510 - 62 - 7 - len(ev.Target()) - 3 - 2)
		if len(out) > maxlen {
			out = out[:maxlen-3]
			out += "..."
		}

		w.Notify(ev.Event, nick, out)
	} else if err != nil {
		w.Notice(nick, err.Error())
	}

	return nil
}

func (_ *Queryer) Google(w irc.Writer, ev *cmd.Event) error {
	q := ev.GetArg("query")
	nick := ev.Nick()
	ev.Close()

	if out, err := query.Google(q); len(out) != 0 {
		out = sanitize(out)
		w.Notify(ev.Event, nick, out)
	} else if err != nil {
		w.Notice(nick, err.Error())
	}

	return nil
}

func (_ *Queryer) Weather(w irc.Writer, ev *cmd.Event) error {
	q := ev.GetArg("query")
	nick := ev.Nick()
	ev.Close()

	if out, err := query.Weather(q, &queryConf); len(out) != 0 {
		out = sanitize(out)
		w.Notify(ev.Event, nick, out)
	} else if err != nil {
		w.Notice(nick, err.Error())
	}

	return nil
}

/* =====================
 Runnable methods.
===================== */

func (r *Runnable) Go(w irc.Writer, ev *cmd.Event) error {
	return sandboxGo(w, ev, "package main\n\nfunc main() {\n%s\n}")
}

func (r *Runnable) Gop(w irc.Writer, ev *cmd.Event) error {
	return sandboxGo(w, ev, "package main\n\nfunc main() {\nfmt.Println(%s)\n}")
}

func sandboxGo(w irc.Writer, ev *cmd.Event, basecode string) error {
	var err error
	var f *os.File

	code := ev.GetArg("code")
	nick := ev.Nick()
	targ := ev.Target()
	ev.Close()

	tmp := os.TempDir()
	frand := rand.Uint32()
	srcfile := filepath.Join(tmp, fmt.Sprintf("%d.go", frand))
	exefile := filepath.Join(tmp, fmt.Sprintf("%d", frand))
	defer os.Remove(srcfile)
	defer os.Remove(exefile)

	if f, err = os.Create(srcfile); err != nil {
		return err
	} else {
		code = code
		_, err = fmt.Fprintf(f, basecode, code)
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}

	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	putStdErr := func(msg string, buf *bytes.Buffer, e error) {
		errMsg := strings.Replace(e.Error(), "\n", "; ", -1)
		outmsg := bytes.Replace(buf.Bytes(), []byte{'\n'}, []byte{';', ' '}, -1)
		w.Notifyf(ev.Event, nick, "\x02go:\x02 %s: %v; %s", msg, errMsg, outmsg)
	}

	goimps := exec.Command("goimports", "-w", srcfile)
	goimps.Stderr = stderr
	if err = goimps.Run(); err != nil {
		putStdErr("Failed to format source", stderr, err)
		return nil
	}
	stderr.Reset()

	build := exec.Command("go", "build", "-o", exefile, srcfile)
	build.Env = os.Environ()
	build.Env = append(build.Env, "GOOS=nacl")
	build.Env = append(build.Env, "GOARCH=amd64p32")
	build.Stderr = stderr
	if err = build.Run(); err != nil {
		putStdErr("Failed to compile", stderr, err)
		return nil
	}
	stderr.Reset()

	run := exec.Command("sel_ldr_x86_64", exefile)
	run.Stderr = stderr
	run.Stdout = stdout
	if err = run.Start(); err != nil {
		putStdErr("Failed to run", stderr, err)
		return nil
	}

	doneChan := make(chan error)
	go func() {
		err := run.Wait()
		doneChan <- err
	}()

	select {
	case err = <-doneChan:
		if err != nil {
			putStdErr("Failed to run", stderr, err)
			return nil
		}
	case <-time.After(time.Second * 4):
		run.Process.Kill()
		w.Notifyf(ev.Event, nick,
			"\x02go:\x02 Program took too long, terminated.")
		return nil
	}

	outbytes := bytes.Replace(stdout.Bytes(), []byte{1}, []byte{}, -1)
	out := fmt.Sprintf("\x02go:\x02 %s", outbytes)
	// ircmaxlen - maxhostsize - PRIVMSG - targetsize - spacing - colons
	maxlen := 2 * (510 - 62 - 7 - len(targ) - 3 - 2)
	if len(out) > maxlen {
		out = out[:maxlen-3]
		out += "..."
	}
	w.Notifyf(ev.Event, nick, out)
	return nil
}

/* =====================
 Handler methods.
===================== */

func (h *Handler) Up(w irc.Writer, ev *cmd.Event) error {
	user := ev.StoredUser
	ch := ev.TargetChannel
	if ch == nil {
		return fmt.Errorf("Must be a channel that the bot is on.")
	}
	chname := ch.Name()

	if !putPeopleUp(ev.Event, chname, user, w) {
		return cmd.MakeFlagsError("ov")
	}
	return nil
}

func (h *Handler) HandleRaw(w irc.Writer, ev *irc.Event) {
	if ev.Name == irc.JOIN {
		h.b.ReadStore(func(s *data.Store) {
			a := s.GetAuthedUser(ev.NetworkID, ev.Sender)
			ch := ev.Target()
			putPeopleUp(ev, ch, a, w)
		})
	}
}

func putPeopleUp(ev *irc.Event, ch string,
	a *data.StoredUser, w irc.Writer) (did bool) {
	if a != nil {
		nick := ev.Nick()
		if a.HasFlag(ev.NetworkID, ch, 'o') {
			w.Sendf("MODE %s +o :%s", ch, nick)
			did = true
		} else if a.HasFlag(ev.NetworkID, ch, 'v') {
			w.Sendf("MODE %s +v :%s", ch, nick)
			did = true
		}
	}
	return
}

func (h *Handler) PrivmsgUser(w irc.Writer, ev *irc.Event) {
	flds := strings.Fields(ev.Message())
	if ev.Nick() == "Aaron" && flds[0] == "do" {
		w.Send(strings.Join(flds[1:], " "))
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var runnable Runnable
	var queryer Queryer
	if conf := query.NewConfig("wolfid.toml"); conf != nil {
		queryConf = *conf
	} else {
		log.Println("Error loading wolfram configuration.")
	}

	qdb, err := quotes.OpenDB("quotes.sqlite3")
	if err != nil {
		log.Fatalln("Error opening quotes db:", err)
	} else {
		qdb.StartServer(":8000")
	}
	defer qdb.Close()
	var quoter = Quoter{qdb}

	err = bot.Run(func(b *bot.Bot) {
		// Quote commands
		b.RegisterCmd(cmd.MkCmd(
			"quote",
			"Retrieves a quote. Randomly selects a quote if no id is provided.",
			"quote",
			&quoter,
			cmd.PRIVMSG, cmd.ALL, "[id]",
		))
		b.RegisterCmd(cmd.MkCmd(
			"quote",
			"Shows the number of quotes in the database.",
			"quotes",
			&quoter,
			cmd.PRIVMSG, cmd.ALL,
		))
		b.RegisterCmd(cmd.MkCmd(
			"quote",
			"Gets the details for a specific quote.",
			"details",
			&quoter,
			cmd.PRIVMSG, cmd.ALL, "id",
		))
		b.RegisterCmd(cmd.MkCmd(
			"quote",
			"Adds a quote to the database.",
			"addquote",
			&quoter,
			cmd.PRIVMSG, cmd.ALL, "quote...",
		))
		b.RegisterCmd(cmd.MkAuthCmd(
			"quote",
			"Removes a quote from the database.",
			"delquote",
			&quoter,
			cmd.PRIVMSG, cmd.ALL, 0, "Q", "id",
		))
		b.RegisterCmd(cmd.MkAuthCmd(
			"quote",
			"Edits an existing quote.",
			"editquote",
			&quoter,
			cmd.PRIVMSG, cmd.ALL, 0, "Q", "id", "quote...",
		))
		b.RegisterCmd(cmd.MkCmd(
			"quote",
			"Shows the address for the quote webserver.",
			"quoteweb",
			&quoter,
			cmd.PRIVMSG, cmd.ALL,
		))

		// Queryer commands
		b.Register(irc.PRIVMSG, &queryer)
		b.RegisterCmd(cmd.MkCmd(
			"query",
			"Submits a query to Google.",
			"google",
			&queryer,
			cmd.PRIVMSG, cmd.ALL, "query...",
		))
		b.RegisterCmd(cmd.MkCmd(
			"query",
			"Submits a query to Wolfram Alpha.",
			"calc",
			&queryer,
			cmd.PRIVMSG, cmd.ALL, "query...",
		))
		b.RegisterCmd(cmd.MkCmd(
			"query",
			"Fetches a weather report from yr.no.",
			"weather",
			&queryer,
			cmd.PRIVMSG, cmd.ALL, "query...",
		))

		// Runnable Commands
		b.RegisterCmd(cmd.MkCmd(
			"runnable",
			"Runs a snippet of sandboxed go code.",
			"go",
			&runnable,
			cmd.PRIVMSG, cmd.ALL, "code...",
		))
		b.RegisterCmd(cmd.MkCmd(
			"runnable",
			"Runs a snippet of sandboxed go code inside fmt.Println().",
			"gop",
			&runnable,
			cmd.PRIVMSG, cmd.ALL, "code...",
		))

		// Handler commands
		handler := Handler{b}
		b.Register(irc.PRIVMSG, &handler)
		b.Register(irc.JOIN, &handler)
		b.RegisterCmd(cmd.MkAuthCmd(
			"simple",
			"Gives the user ops or voice if they have o or v flags respectively.",
			"up",
			&handler,
			cmd.PRIVMSG, cmd.ALL, 0, "", "#chan",
		))
	})

	if err != nil {
		log15.Error(err.Error())
	}
}
