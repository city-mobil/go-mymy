package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go/ioutil2"

	"github.com/city-mobil/go-mymy/internal/util"
)

const (
	saveThreshold int64 = 60 // 1s
)

type position interface {
	fmt.Stringer

	equal(another position) bool
	clone() position
}

type savePos struct {
	pos   position
	force bool
}

type gtidSet struct {
	pos mysql.GTIDSet
}

func newGTIDSet(pos mysql.GTIDSet) *gtidSet {
	return &gtidSet{
		pos: pos,
	}
}

func (g *gtidSet) clone() position {
	return &gtidSet{
		pos: g.pos.Clone(),
	}
}

func (g *gtidSet) equal(another position) bool {
	switch v := another.(type) {
	case *gtidSet:
		return g.pos.Equal(v.pos)
	default:
		return false
	}
}

func (g *gtidSet) String() string {
	if g.pos == nil {
		return ""
	}

	return g.pos.String()
}

func (g *gtidSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		GTID string `json:"gtid"`
	}{
		GTID: g.String(),
	})
}

func (g *gtidSet) UnmarshalJSON(b []byte) error {
	s := struct {
		GTID string `json:"gtid"`
	}{}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	set, err := mysql.ParseGTIDSet(mysql.MySQLFlavor, s.GTID)
	if err != nil {
		return err
	}

	g.pos = set

	return nil
}

type binlogPos struct {
	pos mysql.Position
}

func newBinlogPos(pos mysql.Position) *binlogPos {
	return &binlogPos{
		pos: pos,
	}
}

func (b *binlogPos) clone() position {
	return &binlogPos{
		pos: mysql.Position{
			Name: b.pos.Name,
			Pos:  b.pos.Pos,
		},
	}
}

func (b *binlogPos) equal(another position) bool {
	switch v := another.(type) {
	case *binlogPos:
		return b.pos.Compare(v.pos) == 0
	default:
		return false
	}
}

func (b *binlogPos) String() string {
	return b.pos.String()
}

func (b *binlogPos) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name string `json:"name"`
		Pos  uint32 `json:"pos"`
	}{
		Name: b.pos.Name,
		Pos:  b.pos.Pos,
	})
}

func (b *binlogPos) UnmarshalJSON(src []byte) error {
	s := struct {
		Name string `json:"name"`
		Pos  uint32 `json:"pos"`
	}{}
	if err := json.Unmarshal(src, &s); err != nil {
		return err
	}

	b.pos = mysql.Position{
		Name: s.Name,
		Pos:  s.Pos,
	}

	return nil
}

type stateSaver interface {
	load() (position, error)
	save(pos position, force bool) error
	position() position
	close() error
}

type fileSaver struct {
	pos      position
	gtidMode bool
	filepath string
	savedAt  int64

	mu *sync.RWMutex
}

func newFileSaver(path string, gtidMode bool) (*fileSaver, error) {
	path = util.AbsPath(path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	var pos position
	if gtidMode {
		pos = &gtidSet{pos: emptyGTID}
	} else {
		pos = &binlogPos{pos: mysql.Position{}}
	}

	return &fileSaver{
		pos:      pos,
		gtidMode: gtidMode,
		filepath: path,
		savedAt:  time.Now().Unix(),
		mu:       &sync.RWMutex{},
	}, nil
}

func (s *fileSaver) load() (position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.filepath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if os.IsNotExist(err) {
		return s.pos, nil
	}
	defer func() {
		_ = f.Close()
	}()

	var pos position
	if s.gtidMode {
		pos = &gtidSet{}
	} else {
		pos = &binlogPos{}
	}

	err = json.NewDecoder(f).Decode(&pos)
	if err != nil {
		return nil, err
	}

	s.pos = pos

	return pos, nil
}

func (s *fileSaver) save(pos position, force bool) error {
	if pos == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pos = pos

	now := time.Now().Unix()
	if !force && (now-s.savedAt < saveThreshold) {
		return nil
	}
	s.savedAt = now

	buf, err := json.Marshal(pos)
	if err != nil {
		return fmt.Errorf("failed to save sync position, pos: %s, what: %w", pos, err)
	}

	err = ioutil2.WriteFileAtomic(s.filepath, buf, 0644)
	if err != nil {
		return fmt.Errorf("failed to save sync position, file: %s, pos: %s, what: %w", s.filepath, pos, err)
	}

	return nil
}

func (s *fileSaver) position() position {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.pos == nil {
		return nil
	}

	return s.pos.clone()
}

func (s *fileSaver) close() error {
	return s.save(s.position(), true)
}
