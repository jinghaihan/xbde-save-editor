package save

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileSize = 0x153860
const PartyOffset = 0x152318
const CharacterOffset = 0x152368
const CharacterSize = 0x138
const ArtsOffset = 0x1536e8
const MaximumCurrency = 999999999
const Capacity = 500

type Document struct{ data []byte }
type editError string

func (e editError) Error() string { return string(e) }
func require(ok bool, message string) {
	if !ok {
		panic(editError(message))
	}
}
func Parse(b []byte) (d *Document, err error) {
	defer catch(&err)
	require(len(b) == FileSize, fmt.Sprintf("Expected a %d-byte game save. System saves and thumbnails are not supported.", FileSize))
	d = &Document{bytes.Clone(b)}
	count := int(d.u8(PartyOffset + 24))
	require(count >= 1 && count <= 12, "Unrecognized party layout. This save cannot be edited safely.")
	seen := map[int]bool{}
	for _, id := range d.Party() {
		require(id >= 1 && id <= 27 && !seen[id], "Unrecognized character IDs in the saved party.")
		seen[id] = true
		if id <= 15 {
			l := d.u32(CharacterOffset + (id-1)*CharacterSize)
			require(l >= 1 && l <= 99, fmt.Sprintf("Character %d has an unsupported level or record layout.", id))
		}
	}
	return d, nil
}
func Open(path string) (*Document, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	return Parse(b)
}
func catch(err *error) {
	if p := recover(); p != nil {
		if e, ok := p.(editError); ok {
			*err = e
		} else {
			panic(p)
		}
	}
}

// Edit stages changes so a rejected batch cannot partially change the document.
func (d *Document) Edit(fn func(*Document)) (err error) {
	defer catch(&err)
	staged := &Document{bytes.Clone(d.data)}
	fn(staged)
	d.data = staged.data
	return nil
}
func (d *Document) Bytes() []byte { return bytes.Clone(d.data) }
func (d *Document) SHA256() string {
	s := sha256.Sum256(d.data)
	return strings.ToUpper(hex.EncodeToString(s[:]))
}
func (d *Document) Version() int { return int(d.u32(0)) }
func (d *Document) Party() []int {
	ids := []int{}
	for i := 0; i < int(d.u8(PartyOffset+24)); i++ {
		ids = append(ids, d.u16(PartyOffset+i*2))
	}
	return ids
}
func (d *Document) Campaign() string {
	unknown := true
	for _, id := range d.Party() {
		if id >= 14 {
			return "FutureConnected"
		}
		if id != 1 && id != 7 {
			unknown = false
		}
	}
	if unknown {
		return "Unknown"
	}
	return "MainStory"
}
func (d *Document) MainSupported() bool { return d.Campaign() == "MainStory" && d.Version() == 7 }
func (d *Document) Money() int          { return d.u32(0x151b40) }
func (d *Document) Noponstones() int    { return d.u32(0x10) }
func (d *Document) SetResources(money, noponstones *int) {
	for _, x := range []struct {
		p int
		v *int
	}{{0x151b40, money}, {0x10, noponstones}} {
		if x.v != nil {
			require(*x.v >= 0 && *x.v <= MaximumCurrency, "Currency must be between 0 and 999,999,999.")
			d.w32(x.p, *x.v)
		}
	}
}
func (d *Document) Characters() []Character {
	out := []Character{}
	for _, id := range d.Party() {
		if id <= 15 {
			out = append(out, Character{d, id})
		}
	}
	return out
}
func (d *Document) Character(id int) Character {
	for _, c := range d.Characters() {
		if c.ID == id {
			return c
		}
	}
	panic(editError("Only characters already present in this party can be edited."))
}
func (d *Document) u8(p int) int        { return int(d.data[p]) }
func (d *Document) u16(p int) int       { return int(binary.LittleEndian.Uint16(d.data[p:])) }
func (d *Document) u32(p int) int       { return int(binary.LittleEndian.Uint32(d.data[p:])) }
func (d *Document) w8(p, v int)         { d.data[p] = byte(v) }
func (d *Document) w16(p, v int)        { binary.LittleEndian.PutUint16(d.data[p:], uint16(v)) }
func (d *Document) w32(p, v int)        { binary.LittleEndian.PutUint32(d.data[p:], uint32(v)) }
func (d *Document) bit(p, bit int) bool { return d.data[p+bit/8]&(1<<uint(bit%8)) != 0 }
func (d *Document) setBit(p, bit int)   { d.data[p+bit/8] |= 1 << uint(bit%8) }

type Session struct {
	Path     string
	Document *Document
	original []byte
}

func OpenSession(path string) (*Session, error) {
	p, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	d, e := Open(p)
	if e != nil {
		return nil, e
	}
	return &Session{p, d, d.Bytes()}, nil
}
func (s *Session) Changed() bool { return !bytes.Equal(s.original, s.Document.data) }
func (s *Session) Save(path string) error {
	p, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	if _, e = Parse(s.Document.data); e != nil {
		return e
	}
	same := p == s.Path
	if same {
		b, e := os.ReadFile(s.Path)
		if e != nil {
			return e
		}
		if !bytes.Equal(b, s.original) {
			return fmt.Errorf("the source save changed on disk; reopen it before saving")
		}
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".xbde-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(s.Document.data); e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	if e = os.Rename(f.Name(), p); e != nil {
		return e
	}
	if same {
		s.original = s.Document.Bytes()
	}
	return nil
}
