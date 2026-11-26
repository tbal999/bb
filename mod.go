package main

// Moderation: the admin names moderators; moderators archive (hide) boards and pin them.
// Each moderator's archive list lives in their own .bbmod/mod.json, so a moderator can only
// restore boards they archived themselves.

import (
	"errors"
	"fmt"
	"os"
)

type archived struct {
	BoardRef
	By string // the moderator whose file it is in
}

type moderation struct {
	moderators []string
	archived   []archived
}

func (m moderation) isModerator(user string) bool {
	for _, name := range m.moderators {
		if name == user {
			return true
		}
	}
	return false
}

func (m moderation) isArchived(ref BoardRef) bool {
	for _, x := range m.archived {
		if x.BoardRef == ref {
			return true
		}
	}
	return false
}

// loadMods reads the moderator list from the admin's file and every moderator's archive list.
// Removing someone as a moderator brings back the boards they archived.
func (a *App) loadMods() {
	a.mods = moderation{}
	admin, _ := readModFile(a.dir(a.cfg.Admin, modDir) + modName)
	a.mods.moderators = admin.Moderators
	for _, name := range a.mods.moderators {
		f, err := readModFile(a.dir(name, modDir) + modName)
		if err != nil {
			continue
		}
		for _, ref := range f.Archived {
			if !a.mods.isArchived(ref) {
				a.mods.archived = append(a.mods.archived, archived{ref, name})
			}
		}
	}
}

// changeOwnModFile reads, changes and writes the current user's mod.json.
func (a *App) changeOwnModFile(change func(*modFile)) error {
	path := a.dir(a.user, modDir) + modName
	f, err := readModFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	change(&f)
	if !a.isAdmin() {
		f.Moderators = nil // only the admin's list counts (old files copied it everywhere)
	}
	if err := writeJSON(path, f, 0644); err != nil {
		return err
	}
	a.loadMods()
	return nil
}

func (a *App) archive(ix int) error {
	if !a.isMod() {
		return errors.New("only moderators can archive boards")
	}
	b, err := a.board(ix)
	if err != nil {
		return err
	}
	if err := a.changeOwnModFile(func(f *modFile) { f.Archived = append(f.Archived, b.BoardRef) }); err != nil {
		return err
	}
	fmt.Fprintln(out, b.Title+" archived")
	a.loadBoards()
	return nil
}

// restore un-archives entry n of 'bb mod list'.
func (a *App) restore(n int) error {
	if !a.isMod() {
		return errors.New("only moderators can restore boards")
	}
	if n < 0 || n >= len(a.mods.archived) {
		return fmt.Errorf("there is no archived board %d (see 'bb mod list')", n)
	}
	x := a.mods.archived[n]
	if x.By != a.user {
		return fmt.Errorf("%s was archived by %s - only they can restore it", x.Title, x.By)
	}
	err := a.changeOwnModFile(func(f *modFile) {
		var keep []BoardRef
		for _, ref := range f.Archived {
			if ref != x.BoardRef {
				keep = append(keep, ref)
			}
		}
		f.Archived = keep
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(out, x.Title+" restored")
	a.loadBoards()
	return nil
}

func (a *App) printArchived() {
	if len(a.mods.archived) == 0 {
		fmt.Fprintln(out, "no archived boards")
	}
	for i, x := range a.mods.archived {
		fmt.Fprintf(out, "%d) %s | %s | archived by %s\n", i, x.Title, x.Date, x.By)
	}
}

// setModerator adds or removes a moderator (admin only).
func (a *App) setModerator(user string, on bool) error {
	if !a.isAdmin() {
		return errors.New("only the admin can add or remove moderators")
	}
	return a.changeOwnModFile(func(f *modFile) {
		var keep []string
		for _, name := range f.Moderators {
			if name != user {
				keep = append(keep, name)
			}
		}
		if on {
			keep = append(keep, user)
		}
		f.Moderators = keep
	})
}

// initialise creates the shared files (admin only, once). Running it again is harmless.
func (a *App) initialise() error {
	if !a.isAdmin() {
		return errors.New("you tried to initiate bb - but you are not the admin. The admin is: " + a.cfg.Admin)
	}
	for _, name := range []string{anonName, pinName} {
		path := a.sharedFile(name)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(path, []byte("{}"), 0666); err != nil {
				return err
			}
		}
		if err := os.Chmod(path, 0666); err != nil { // everyone posts anonymously and pins through these
			return err
		}
	}
	if !a.isMod() {
		if err := a.setModerator(a.user, true); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "the files have now either been created & bb is initiated - or they already existed!")
	return nil
}

// ---- pins ----

func (a *App) loadPins() {
	data, err := readShared(a.sharedFile(pinName))
	if err != nil {
		a.pins = nil
		return
	}
	a.pins, _ = parsePins(data)
}

func (a *App) isPinned(ref BoardRef) bool {
	for _, p := range a.pins {
		if p == ref {
			return true
		}
	}
	return false
}

// setPin pins or unpins board ix (moderators only).
func (a *App) setPin(ix int, on bool) error {
	if !a.isMod() {
		return errors.New("only moderators can pin boards")
	}
	b, err := a.board(ix)
	if err != nil {
		return err
	}
	err = updateShared(a.sharedFile(pinName), func(old []byte) ([]byte, error) {
		pins, err := parsePins(old)
		if err != nil {
			return nil, fmt.Errorf("could not read the pin file: %w", err)
		}
		var keep []BoardRef
		for _, p := range pins {
			if p != b.BoardRef {
				keep = append(keep, p)
			}
		}
		if on {
			keep = append(keep, b.BoardRef)
		}
		return formatPins(keep)
	})
	a.loadPins()
	return err
}
