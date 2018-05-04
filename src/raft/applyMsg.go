package raft

import (
	"fmt"
	"sort"
)

// ApplyMsg 是发送消息
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
	CommandIndex int         // Command zd Raft.logs 中的索引号
	Command      interface{} // Command 的具体内容
}

func (a ApplyMsg) String() string {
	return fmt.Sprintf("ApplyMsg[idx:%d, cmd:%v]", a.CommandIndex, a.Command)
}

// 每当 rf.logs 或 rf.commitIndex 有变化时，就收到通知
// 然后，检查发现有可以 commit 的 entry 的话
// 就通过 applyCh 发送 ApplyMsg 给 replication state machine 进行 commit
func (rf *Raft) checkApplyLoop(applyCh chan ApplyMsg) {
	rf.shutdownWG.Add(1)
	for {

		select {
		case <-rf.toCheckApplyChan:
			debugPrintf(" S#%d 在 checkApplyLoop 的 case <- rf.toCheckApplyChan，收到信号。将要检查是否有新的 entry 可以 commig", rf.me)
		case <-rf.shutdownChan:
			debugPrintf(" S#%d 在 checkApplyLoop 的 case <- rf.shutdownChan，收到信号。关闭 checkApplyLoop", rf.me)
			rf.shutdownWG.Done()
			return
		}

		// update rf.commitIndex based on matchIndex[]
		// if there exists an N such that N > commitIndex, a majority of matchIndex[i] >= N
		// and log[N].term == currentTerm:
		// set commitIndex = N
		if rf.state == LEADER {
			// 先获取的自己的 matchIndex
			rf.matchIndex[rf.me] = len(rf.logs) - 1
			// 然后统计
			mmIndex := maxMajorityIndex(rf.matchIndex)

			// find the max matchIndex committed
			// paper 5.4.2, only log entries from the leader's current term are committed by counting replicas
			if mmIndex > rf.commitIndex &&
				rf.logs[mmIndex].LogTerm == rf.currentTerm {
				rf.commitIndex = mmIndex
			}
			debugPrintf("%s , maxMajorityIndex:%d, rf.commitIndex:%d", rf, mmIndex, rf.commitIndex)
		}

		if rf.lastApplied == rf.commitIndex {
			continue
		}

		debugPrintf("%s lastApplied: %d, commitIndex: %d", rf, rf.lastApplied, rf.commitIndex)
		for rf.lastApplied < rf.commitIndex {
			rf.lastApplied++
			applyMsg := ApplyMsg{
				CommandValid: true,
				Command:      rf.logs[rf.lastApplied].Command,
				CommandIndex: rf.lastApplied}
			debugPrintf("%s apply %s", rf, applyMsg)
			applyCh <- applyMsg
		}

		// persist only when possible committed data
		// for leader, it's easy to determine
		// persist leader during commit
		if rf.state == LEADER {
			rf.persist()
		}

	}
}

// 返回 matchIndex 中超过半数的 Index
// 例如
// matchIndex == {8,7,6,5,4}
// 	     temp == {4,5,6,7,8}
// i = (5-1)/2 = 2
// 超过半数的 server 拥有 {4,5,6}
// 其中 temp[i] == 6 是最大值
func maxMajorityIndex(matchIndex []int) int {
	temp := make([]int, len(matchIndex))
	copy(temp, matchIndex)

	sort.Ints(temp)

	i := (len(matchIndex) - 1) / 2

	return temp[i]
}
