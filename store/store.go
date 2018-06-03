package store

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Rec struct {
	Fname string
	Exp   time.Time
}

type Store struct {
	sync.RWMutex
	Store map[string]Rec
}

func NewStore() *Store {
	s := &Store{Store: make(map[string]Rec)}
	s.Load()
	go s.Expire()
	return s
}

func (s *Store) Set(fname string, days int) string {
	s.Lock()
	defer s.Unlock()

	buf := make([]byte, 12)
	rand.Read(buf)

	k := base64.URLEncoding.EncodeToString(buf)

	s.Store[k] = Rec{
		Fname: fname,
		Exp:   time.Now().AddDate(0, 0, days),
	}
	return k
}

func (s *Store) Get(key string) (string, error) {
	s.RLock()
	defer s.RUnlock()
	if r, ok := s.Store[key]; ok {
		return r.Fname, nil
	}
	return "", fmt.Errorf("Not found")
}

func (s *Store) Expire() {
	for {
		s.Lock()
		for k, v := range s.Store {
			if v.Exp.Before(time.Now()) {
				delete(s.Store, k)
			}
		}
		s.Save()
		s.Unlock()
		time.Sleep(time.Hour)
	}
}

func (s *Store) Save() {
	f, err := os.Create("store.json")
	if err != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("","    ")
	enc.Encode(s)
}

func (s *Store) Load() {
	f, err := os.Open("store.json")
	if err != nil {
		return
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.Decode(s)
}
