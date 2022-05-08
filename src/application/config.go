package application

import (
	// "GoPaddle-Raft/kvraft"
	"GoPaddle-Raft/labrpc"
	"GoPaddle-Raft/raft"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/data/binding"
)

func randstring(n int) string {
	b := make([]byte, 2*n)
	crand.Read(b)
	s := base64.URLEncoding.EncodeToString(b)
	return s[0:n]
}

func makeSeed() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := crand.Int(crand.Reader, max)
	x := bigx.Int64()
	return x
}

func (cfg *Config) ConnectAll() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for i := 0; i < cfg.n; i++ {
		cfg.connectUnlocked(i, cfg.All())
	}
}

func (cfg *Config) disconnect(i int, from []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.disconnectUnlocked(i, from)
}

func (cfg *Config) All() []int {
	all := make([]int, cfg.n)
	for i := 0; i < cfg.n; i++ {
		if cfg.Kvservers[i] != nil && cfg.Kvservers[i].disconn {
			continue
		}
		all[i] = i
	}
	return all
}

// attach server i to servers listed in to
// caller must hold Cfg.mu
func (cfg *Config) connectUnlocked(i int, to []int) {
	// log.Printf("connect peer %d to %v\n", i, to)

	// outgoing socket files
	for j := 0; j < len(to); j++ {
		endname := cfg.endnames[i][to[j]]
		cfg.net.Enable(endname, true)
	}

	// incoming socket files
	for j := 0; j < len(to); j++ {
		endname := cfg.endnames[to[j]][i]
		cfg.net.Enable(endname, true)
	}
}

func (cfg *Config) connect(i int, to []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.connectUnlocked(i, to)
}

// Randomize server handles
func random_handles(kvh []*labrpc.ClientEnd) []*labrpc.ClientEnd {
	sa := make([]*labrpc.ClientEnd, len(kvh))
	copy(sa, kvh)
	for i := range sa {
		j := rand.Intn(i + 1)
		sa[i], sa[j] = sa[j], sa[i]
	}
	return sa
}

// caller should hold cfg.mu
func (cfg *Config) ConnectClientUnlocked(ck *Clerk, to []int) {
	// log.Printf("ConnectClient %v to %v\n", ck, to)
	endnames := cfg.clerks[ck]
	for j := 0; j < len(to); j++ {
		s := endnames[to[j]]
		cfg.net.Enable(s, true)
	}
}

func (cfg *Config) Op() {
	atomic.AddInt32(&cfg.ops, 1)
}

// Shutdown a server by isolating it
func (cfg *Config) ShutdownServer(i int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	cfg.disconnectUnlocked(i, cfg.All())

	// disable client connections to the server.
	// it's important to do this before creating
	// the new Persister in saved[i], to avoid
	// the possibility of the server returning a
	// positive reply to an Append but persisting
	// the result in the superseded Persister.
	cfg.net.DeleteServer(i)

	// a fresh persister, in case old instance
	// continues to update the Persister.
	// but copy old persister's content so that we always
	// pass Make() the last persisted state.
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	}

	kv := cfg.Kvservers[i]
	if kv != nil {
		cfg.mu.Unlock()
		kv.Kill()
		cfg.mu.Lock()
		cfg.Kvservers[i] = nil
	}
}

func (cfg *Config) disconnectUnlocked(i int, from []int) {
	// log.Printf("disconnect peer %d from %v\n", i, from)

	// outgoing socket files
	for j := 0; j < len(from); j++ {
		if cfg.endnames[i] != nil {
			endname := cfg.endnames[i][from[j]]
			cfg.net.Enable(endname, false)
		}
	}

	// incoming socket files
	for j := 0; j < len(from); j++ {
		if cfg.endnames[j] != nil {
			endname := cfg.endnames[from[j]][i]
			cfg.net.Enable(endname, false)
		}
	}
}

func (cfg *Config) op() {
	atomic.AddInt32(&cfg.ops, 1)
}

type Config struct {
	mu           sync.Mutex
	net          *labrpc.Network
	n            int
	Kvservers    []*KVServer
	saved        []*raft.Persister
	endnames     [][]string // names of each server's sending ClientEnds
	clerks       map[*Clerk][]string
	nextClientId int
	maxraftstate int
	start        time.Time // time at which make_config() was called
	// begin()/end() statistics

	t0             time.Time // time at which test_test.go called cfg.begin()
	rpcs0          int       // rpcTotal() at start of test
	ops            int32     // number of clerk get/put/append method calls
	ConsoleBinding binding.ExternalStringList
}

func (cfg *Config) ShowServerInfo() []*KVServer {
	return cfg.Kvservers
}

var ncpu_once sync.Once

func Make_config(n int, unreliable bool, maxraftstate int) *Config {
	ncpu_once.Do(func() {
		if runtime.NumCPU() < 2 {
			fmt.Printf("warning: only one CPU, which may conceal locking bugs\n")
		}
		rand.Seed(makeSeed())
	})
	runtime.GOMAXPROCS(4)
	cfg := &Config{}
	cfg.net = labrpc.MakeNetwork()
	cfg.n = n
	cfg.Kvservers = make([]*KVServer, cfg.n)
	cfg.saved = make([]*raft.Persister, cfg.n)
	cfg.endnames = make([][]string, cfg.n)
	cfg.clerks = make(map[*Clerk][]string)
	cfg.nextClientId = cfg.n + 1000 // client ids start 1000 above the highest serverid
	cfg.maxraftstate = maxraftstate
	cfg.start = time.Now()

	tmpStr := time.Now().Format("2006/01/02 15:04:05 ") + "[GoPaddle]: Platform has been started.\n"
	cfg.ConsoleBinding = binding.BindStringList(
		&[]string{tmpStr},
	)
	// create a full set of KV servers.
	for i := 0; i < cfg.n; i++ {
		cfg.StartServer(i)
	}

	//.Printf("Servers: %v\n", Cfg.Kvservers)

	cfg.ConnectAll()

	cfg.net.Reliable(!unreliable)

	return cfg
}

// If restart servers, first call ShutdownServer
func (cfg *Config) StartServer(i int) {
	cfg.mu.Lock()
	// a fresh set of outgoing ClientEnd names.
	cfg.endnames[i] = make([]string, cfg.n)
	for j := 0; j < cfg.n; j++ {
		cfg.endnames[i][j] = randstring(20)
	}

	// a fresh set of ClientEnds.
	ends := make([]*labrpc.ClientEnd, cfg.n)
	for j := 0; j < cfg.n; j++ {
		ends[j] = cfg.net.MakeEnd(cfg.endnames[i][j])
		cfg.net.Connect(cfg.endnames[i][j], j)
	}

	// a fresh persister, so old instance doesn't overwrite
	// new instance's persisted state.
	// give the fresh persister a copy of the old persister's
	// state, so that the spec is that we pass StartKVServer()
	// the last persisted state.
	if cfg.saved[i] != nil {
		cfg.saved[i] = cfg.saved[i].Copy()
	} else {
		cfg.saved[i] = raft.MakePersister()
	}
	cfg.mu.Unlock()

	cfg.Kvservers[i] = StartKVServer(ends, i, cfg.saved[i], cfg.maxraftstate, cfg.ConsoleBinding)

	kvsvc := labrpc.MakeService(cfg.Kvservers[i])
	rfsvc := labrpc.MakeService(cfg.Kvservers[i].Rf)
	srv := labrpc.MakeServer()
	srv.AddService(kvsvc)
	srv.AddService(rfsvc)
	cfg.net.AddServer(i, srv)
}

// Create a Clerk with Clerk specific server names.
// Give it connections to all of the servers, but for
// now enable only connections to servers in to[].
func (cfg *Config) MakeClient(to []int) *Clerk {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// a fresh set of ClientEnds.
	ends := make([]*labrpc.ClientEnd, cfg.n)
	endnames := make([]string, cfg.n)
	for j := 0; j < cfg.n; j++ {
		endnames[j] = randstring(20)
		ends[j] = cfg.net.MakeEnd(endnames[j])
		cfg.net.Connect(endnames[j], j)
	}
	//fmt.Printf("Client ends: %v\n", ends)
	//fmt.Printf("endnames: %v\n", endnames)

	ck := MakeClerk(random_handles(ends))
	cfg.clerks[ck] = endnames
	cfg.nextClientId++
	cfg.ConnectClientUnlocked(ck, to)
	return ck
}

func (cfg *Config) deleteClient(ck *Clerk) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	v := cfg.clerks[ck]
	for i := 0; i < len(v); i++ {
		os.Remove(v[i])
	}
	delete(cfg.clerks, ck)
}

func (cfg *Config) cleanup() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for i := 0; i < len(cfg.Kvservers); i++ {
		if cfg.Kvservers[i] != nil {
			cfg.Kvservers[i].Kill()
		}
	}
	cfg.net.Cleanup()
}

// Sets up 2 partitions with connectivity between servers in each  partition.
func (cfg *Config) partition(p1 []int, p2 []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	// log.Printf("partition servers into: %v %v\n", p1, p2)
	for i := 0; i < len(p1); i++ {
		cfg.disconnectUnlocked(p1[i], p2)
		cfg.connectUnlocked(p1[i], p1)
	}
	for i := 0; i < len(p2); i++ {
		cfg.disconnectUnlocked(p2[i], p1)
		cfg.connectUnlocked(p2[i], p2)
	}
}
