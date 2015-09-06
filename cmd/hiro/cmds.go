package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/felixge/hiro/datetime"
	"github.com/felixge/hiro/db"
	"github.com/felixge/hiro/term"
)

func cmdStart(d db.DB, categoryString string) {
	entries, err := active(d)
	if err != nil {
		fatal(err)
	}
	now := time.Now()
	category := splitCategory(categoryString)
	entry := &db.Entry{Category: category, Start: now}
	if err := d.Save(entry); err != nil {
		fatal(err)
	} else if err := FprintEntry(os.Stdout, entry, PrintHideDuration|PrintHideEnd); err != nil {
		fatal(err)
	} else if err := endAt(d, entries, now); err != nil {
		fatal(err)
	}
}

func cmdEnd(d db.DB) {
	if entries, err := active(d); err != nil {
		fatal(err)
	} else if err := endAt(d, entries, time.Now()); err != nil {
		fatal(err)
	}
}

func active(d db.DB) ([]*db.Entry, error) {
	if itr, err := d.Query(db.Query{Active: true}); err != nil {
		return nil, err
	} else {
		return db.IteratorEntries(itr)
	}
}

func endAt(d db.DB, entries []*db.Entry, t time.Time) error {
	for _, entry := range entries {
		entry.End = t
		if err := d.Save(entry); err != nil {
			return err
		} else if err := FprintEntry(os.Stdout, entry, PrintDefault); err != nil {
			return err
		}
	}
	return nil
}

func cmdList(d db.DB) {
	itr, err := d.Query(db.Query{})
	if err != nil {
		fatal(err)
	} else if err := FprintIterator(os.Stdout, itr, PrintDefault); err != nil {
		fatal(err)
	}
}

func cmdEdit(d db.DB, id string) {
	itr, err := d.Query(db.Query{IDs: []string{id}})
	if err != nil {
		fatal(err)
	}
	e := term.NewEditor()
	if err := FprintIterator(e, itr, PrintSeparator|PrintHideDuration); err != nil {
		fatal(err)
	} else if err := e.Run(); err != nil {
		fatal(err)
	} else if entries, err := ParseEntries(e); err != nil {
		fatal(err)
	} else if l := len(entries); l == 0 {
		return
	} else if l > 1 {
		fatal(fmt.Errorf("editing multiple entries is not supported yet"))
	} else if err := d.Save(entries[0]); err != nil {
		fatal(err)
	} else if err := FprintIterator(os.Stdout, db.EntryIterator(entries), PrintDefault); err != nil {
		fatal(err)
	}
}

func cmdSummary(d db.DB, durationS, firstDayS string, asc bool) {
	duration, err := datetime.ParseDuration(durationS)
	if err != nil {
		fatal(err)
	}
	firstDay, err := datetime.ParseWeekday(firstDayS)
	if err != nil {
		fatal(err)
	}
	entries, err := d.Query(db.Query{Asc: asc})
	if err != nil {
		fatal(err)
	}
	var (
		now        = time.Now()
		entry      *db.Entry
		durations  *datetime.Iterator
		fromTo     [2]time.Time
		categories map[string]time.Duration
	)
	for {
		entry, err = entries.Next()
		if err == io.EOF {
			if _, err := fmt.Printf(FormatSummary(categories)); err != nil {
				fatal(err)
			}
			break
		} else if err != nil {
			fatal(err)
		}
		if durations == nil {
			durations = datetime.NewIterator(entry.Start, duration, asc, firstDay)
		}
		if fromTo[0].IsZero() || entry.Start.Before(fromTo[0]) {
			if _, err := fmt.Printf(FormatSummary(categories)); err != nil {
				fatal(err)
			}
			fromTo[0], fromTo[1] = durations.Next()
			categories = make(map[string]time.Duration)
			if _, err := fmt.Printf("%s\n\n", FormatSummaryHeadline(fromTo[0], fromTo[1], duration)); err != nil {
				fatal(err)
			}
		}
		partialDuration := entry.PartialDuration(now, fromTo[0], fromTo[1])
		if partialDuration > 0 {
			name := strings.Join(entry.Category, ":")
			categories[name] += partialDuration
		}
	}
}

func cmdVersion() {
	fmt.Printf("%s\n", version)
}
