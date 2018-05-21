package blacklist

import (
	"log"
	"sync"
	"time"
)

const banCnt = 5
const banHrs = 24
const banAll = 100

type BRec struct {
	cnt int
	la  time.Time
}

type BlackList struct {
	sync.Mutex
	m      map[string]BRec
	allcnt int
}

func NewBlackList() *BlackList {
	l := &BlackList{m: make(map[string]BRec)}
	go l.Clear()
	return l
}

func (l *BlackList) IsBlack(ip string) bool {
	l.Lock()
	defer l.Unlock()
	if l.allcnt >= banAll {
		return true
	}
	if b, ok := l.m[ip]; ok {
		if b.cnt >= banCnt {
			return true
		}
	}
	return false
}

func (l *BlackList) PaintBlack(ip string) bool {
	l.Lock()
	defer l.Unlock()
	l.allcnt++
	if b, ok := l.m[ip]; ok {
		if b.cnt >= banCnt {
			return true
		} else {
			b.cnt++
			b.la = time.Now()
			l.m[ip] = b
		}
	} else {
		l.m[ip] = BRec{cnt: 1, la: time.Now()}
	}
	return l.allcnt >= banAll

}

func (l *BlackList) Clear() {
	tick := time.Tick(time.Hour)
	for range tick {
		l.Lock()
		for k, v := range l.m {
			// ban one day
			log.Println("Banned", k)
			if v.cnt >= banCnt && time.Now().After(v.la.Add(banHrs*time.Hour)) {
				delete(l.m, k)
			}
		}
		l.allcnt = 0
		l.Unlock()
	}
}
