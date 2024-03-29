package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"6.824-2018-kyrie/src/labgob"
	"bytes"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)
import "6.824-2018-kyrie/src/labrpc"

// import "bytes"
// import "labgob"

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

const NULL int = -1

type Log struct {
	Term    int
	Command interface{}
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	state State

	//Persistent state on all servers
	currentTerm int   "latest term server has seen(initialized to 0 on first boot,increases monotonically)"
	votedFor    int   "candidateId that received vote in current term(or null if none)"
	log         []Log "log entries;each entry contains command for state machine,and term when entry was received by leader(first index is 1)"

	//Volatile state on all servers
	commitIndex int "index of highest log entry known to be committed(initialized to 0,increases monotonically)"
	lastApplied int "index of highest log entry applied to state machine(initialized to 0,increases monotonically)"

	//Volatile state on leaders(Reinitialized after election)
	nextIndex  []int "for each server,index of the next log entry to send to that server(initialized to leader last log index +1)"
	matchIndex []int "for each server,index of highest log entry known to be replicated on server(initialized to 0,increases monotonically)"

	applyCh     chan ApplyMsg
	voteCh      chan bool
	appendLogCh chan bool

	killCh chan bool
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.state == Leader
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []Log
	if d.Decode(&currentTerm) != nil || d.Decode(&votedFor) != nil || d.Decode(&log) != nil {
		DPrintf("readPersist ERROR for server %v", rf.me)
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.log = log
	}
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

//
// RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term > rf.currentTerm {
		rf.beFollower(args.Term)
	}

	reply.Term = rf.currentTerm
	reply.VoteGranted = false

	if (args.Term < rf.currentTerm) || (rf.votedFor != NULL && rf.votedFor != args.CandidateId) {

	} else if args.LastLogTerm < rf.getLastLogTerm() || (args.LastLogTerm == rf.getLastLogTerm() && args.LastLogIndex < rf.getLastLogIdx()) {

	} else {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		rf.state = Follower
		rf.persist()
		send(rf.voteCh)
	}
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

type AppendEntriesArgs struct {
	Term         int   "leader's term"
	LeaderId     int   "so follower can redirect clients"
	PrevLogIndex int   "index of log entry immediately preceding"
	PrevLogTerm  int   "term of prevLogIndex entry"
	Entries      []Log "log entries to store(empty for heartbeat;may send more than one for efficiency)"
	LeaderCommit int   "leader's commitIndex"
}

type AppendEntriesReply struct {
	Term          int  "currentTerm,for leader to update itself"
	Success       bool "return true if follower contained entry matching prevLogIndex and prevLogTerm"
	ConflictIndex int  "first Log index it stores for that conflict term"
	ConflictTerm  int  "the term of the conflicting log entry"
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	index := -1
	term := rf.currentTerm
	isLeader := rf.state == Leader

	// Your code here (2B).
	if isLeader {
		index = rf.getLastLogIdx() + 1
		newLog := Log{
			Term:    rf.currentTerm,
			Command: command,
		}
		rf.log = append(rf.log, newLog)
		rf.persist()
	}
	return index, term, isLeader
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
	send(rf.killCh)
}

func (rf *Raft) beCandidate() {
	rf.state = Candidate  //switch to Candidate
	rf.currentTerm++      //increment currentTerm
	rf.votedFor = rf.me   // vote myself first
	rf.persist()          //save to storage
	go rf.startElection() //send RequestVote RPCs to all other servers
}

//If election timeout elapses:start new election handled in caller
func (rf *Raft) startElection() {
	rf.mu.Lock()
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.getLastLogIdx(),
		LastLogTerm:  rf.getLastLogTerm(),
	}
	rf.mu.Unlock()
	var votes int32 = 1
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		go func(idx int) {
			reply := &RequestVoteReply{}
			ret := rf.sendRequestVote(idx, &args, reply)

			if ret {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				if reply.Term > rf.currentTerm {
					rf.beFollower(reply.Term)
					return
				}
				if rf.state != Candidate || rf.currentTerm != args.Term {
					return
				}
				if reply.VoteGranted {
					atomic.AddInt32(&votes, 1)
				}
				if atomic.LoadInt32(&votes) > int32(len(rf.peers)/2) {
					rf.beLeader()
					rf.startAppendLog()
					send(rf.voteCh)
				}
			}
		}(i)
	}
}

func (rf *Raft) getLastLogIdx() int {
	return len(rf.log) - 1
}

func (rf *Raft) getLastLogTerm() int {
	idx := rf.getLastLogIdx()
	if idx < 0 {
		return -1
	}
	return rf.log[idx].Term
}

func (rf *Raft) beLeader() {
	if rf.state != Candidate {
		return
	}
	rf.state = Leader
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for i := 0; i < len(rf.nextIndex); i++ {
		rf.nextIndex[i] = rf.getLastLogIdx() + 1
	}
}

func (rf *Raft) beFollower(termId int) {
	rf.state = Follower
	rf.votedFor = NULL
	rf.currentTerm = termId
	rf.persist()
}

//AppendEntries RPC Handler
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term > rf.currentTerm {
		rf.beFollower(args.Term)
	}
	send(rf.appendLogCh)

	reply.Term = rf.currentTerm
	reply.Success = false
	reply.ConflictTerm = NULL
	reply.ConflictIndex = 0
	prevLogIdxTerm := -1
	logSize := len(rf.log)
	if args.PrevLogIndex >= 0 && args.PrevLogIndex < len(rf.log) {
		prevLogIdxTerm = rf.log[args.PrevLogIndex].Term
	}
	if prevLogIdxTerm != args.PrevLogTerm {
		reply.ConflictIndex = logSize
		if prevLogIdxTerm == -1 {

		} else {
			reply.ConflictTerm = prevLogIdxTerm
			for i := 0; i < logSize; i++ {
				if rf.log[i].Term == reply.ConflictTerm {
					reply.ConflictIndex = i
					break
				}
			}
		}
		return
	}

	//Reply false if args.term < currentTerm
	if args.Term < rf.currentTerm {
		return
	}

	index := args.PrevLogIndex
	for i := 0; i < len(args.Entries); i++ {
		index++
		if index < logSize {
			if rf.log[index].Term == args.Entries[i].Term {
				continue
			} else {
				rf.log = rf.log[:index]
			}
		}
		rf.log = append(rf.log, args.Entries[i:]...)
		rf.persist()
		break
	}

	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = Min(args.LeaderCommit, rf.getLastLogIdx())
		rf.updateLastApplied()
	}
	reply.Success = true
}

//Leader发出AppendLog动作
func (rf *Raft) startAppendLog() {
	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		go func(idx int) {
			for {
				rf.mu.Lock()
				if rf.state != Leader {
					rf.mu.Unlock()
					return
				}
				args := AppendEntriesArgs{
					Term:         rf.currentTerm,
					LeaderId:     rf.me,
					PrevLogIndex: rf.getPrevLogIdx(idx),
					PrevLogTerm:  rf.getPrevLogTerm(idx),
					Entries:      append(make([]Log, 0), rf.log[rf.nextIndex[idx]:]...),
					LeaderCommit: rf.commitIndex,
				}
				rf.mu.Unlock()
				reply := &AppendEntriesReply{}
				ret := rf.sendAppendEntries(idx, &args, reply)
				rf.mu.Lock()
				if !ret || rf.state != Leader || rf.currentTerm != args.Term {
					rf.mu.Unlock()
					return
				}
				if reply.Term > rf.currentTerm {
					rf.beFollower(reply.Term)
					rf.mu.Unlock()
					return
				}
				if reply.Success { //If AppendEntries success,update nextIndex and matchIndex for each follower
					rf.updateNextMatchIdx(idx, args.PrevLogIndex+len(args.Entries))
					rf.mu.Unlock()
					return
				} else { // If AppendEntries RPC fails because of log entry's inconsistency,retry after decrement nextIndex
					targetIndex := reply.ConflictIndex
					if reply.ConflictTerm != NULL {
						logSize := len(rf.log)
						for i := 0; i < logSize; i++ {
							if rf.log[i].Term != reply.ConflictTerm {
								continue
							}
							for i < logSize && rf.log[i].Term == reply.ConflictTerm {
								i++
							}
							targetIndex = i
						}
					}
					rf.nextIndex[idx] = Min(len(rf.log), targetIndex)
					rf.mu.Unlock()
				}
			}
		}(i)
	}
}

func (rf *Raft) getPrevLogIdx(idx int) int {
	return rf.nextIndex[idx] - 1
}

func (rf *Raft) getPrevLogTerm(idx int) int {
	prevLogIdx := rf.getPrevLogIdx(idx)
	if prevLogIdx < 0 {
		return -1
	}
	return rf.log[prevLogIdx].Term
}

func (rf *Raft) sendAppendEntries(i int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[i].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) updateLastApplied() {
	for rf.lastApplied < rf.commitIndex {
		rf.lastApplied++
		curLog := rf.log[rf.lastApplied]
		applyMsg := ApplyMsg{
			CommandValid: true,
			Command:      curLog.Command,
			CommandIndex: rf.lastApplied,
		}
		rf.applyCh <- applyMsg
	}
}

func (rf *Raft) updateNextMatchIdx(server int, matchIdx int) {
	rf.matchIndex[server] = matchIdx
	rf.nextIndex[server] = matchIdx + 1
	rf.updateCommitIndex()
}

func (rf *Raft) updateCommitIndex() {
	rf.matchIndex[rf.me] = len(rf.log) - 1
	copyMatchIndex := make([]int, len(rf.matchIndex))
	copy(copyMatchIndex, rf.matchIndex)
	sort.Sort(sort.Reverse(sort.IntSlice(copyMatchIndex)))      //copyMatchIndex中元素降序排序,按照Raft协议中每个节点中的matchIndex进行降序排序
	N := copyMatchIndex[len(copyMatchIndex)/2]                  //取copyMatchIndex数组中中间节点元素，也就是matchIndex数组中的元素：对于每一个服务器节点，记录已经复制到该服务器的log entry的最高索引值
	if N > rf.commitIndex && rf.log[N].Term == rf.currentTerm { //updateCommitIndex的关键就是N>commitIndex，a majority of matchIndex[i]≥N。如果超过一半的服务器节点已经
		rf.commitIndex = N
		rf.updateLastApplied() //commited后需要apply到上层状态机，需要更新lastApplied
	}
}

func send(ch chan bool) {
	select {
	case <-ch: //do nothing,just for consume element in channel
	default:
	}
	ch <- true
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.state = Follower //all raft server's initial state is follower
	rf.currentTerm = 0
	rf.votedFor = NULL
	rf.log = make([]Log, 1)

	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(peers))  //create an len(peers) slice for store nextIndex for every raft server
	rf.matchIndex = make([]int, len(peers)) //create an len(peers) slice for store matchIndex for every raft server

	rf.applyCh = applyCh
	rf.voteCh = make(chan bool, 1)
	rf.appendLogCh = make(chan bool, 1)
	rf.killCh = make(chan bool, 1)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	heartbeatTime := time.Duration(100) * time.Millisecond

	go func() {
		for {
			select {
			case <-rf.killCh:
				return
			default:

			}
			electionTime := time.Duration(rand.Intn(100)+300) * time.Millisecond
			switch rf.state {
			case Follower, Candidate:
				select {
				case <-rf.voteCh:
				case <-rf.appendLogCh:
				case <-time.After(electionTime):
					rf.mu.Lock()
					rf.beCandidate() // turn to candidate,send RequestVoteRpc to other raft servers,start election
					rf.mu.Unlock()
				}
			case Leader:
				time.Sleep(heartbeatTime)
				rf.startAppendLog()
			}
		}
	}()

	return rf
}
