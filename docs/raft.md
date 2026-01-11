
# Raft - algorithm
Here is my kind of Raft explanation.

Soooo, in raft we have 3 states:
- Follower
- Candidate
- Leader

## Follower
A follower is a passive state. It just listens for `Heartbeats` from the leader. If it doesn't receive a heartbeat within a certain timeout, it assumes that there is no leader and transitions to the candidate state.

## Candidate
Intermidiate state. When a follower becomes a candidate, it starts an election by incrementing its term and sending `RequestVote` to other nodes. If it receives votes from a majority of nodes, it becomes the leader. If it receives a heartbeat from a valid leader, it reverts back to follower.

## Leader
The leader is the active state. It handles all client requests and manages log replication. It sends periodic heartbeats to followers to maintain authority. If a leader fails, followers will timeout and start a new election.

That is basically how Raft works, but edge cases are in my view the most important part as the idea is pretty simple.

## Notation
So for sake of simplicity i will use this latex syntax to represent a node state:
- $C_{index}^{term}$ - Candidate node with index and term
- $F_{index}^{term}$ - Follower node with index and term
- $L_{index}^{term}$ - Leader node with index and term

# Assumptions
Raft makes some assumptions to ensure correctness:
- Nodes can fail and recover
- Network can lose or delay messages (no reordering or corruption)
- There is a majority of non-faulty nodes (more than half)
- There is no byzantine behavior (nodes do not act maliciously)
- N is odd (to avoid ties in elections) we might have pretty ugly network partition 50/50 which would stop the system from making any progress.

# Raft - edge cases
Leader broadcasts log entries to followers. Followers append these entries and everything is fine. Problems arise when two things happen:
1. Node failures - some nodes go down or become unreachable
2. Concurrent elections - two nodes become candidates at the same time
> Other specific cases can be derived from these two main issues (i hope). Like when leader dies it is network partition + crash.

Raft handles these issues and i will try to explain how each edge case is handled.

## 1. Nodes unreachable
When a node (in this case follower) becomes unreachable. And it can be cause by either some nodes crash or network partition. It does not make any difference so we will treat both cases the same way.

Network partition - means that part of the network disconnects from the rest. So we have two groups (it can be extended to any amount basically). One group has majority of nodes, others don't.

### Disconnection of some nodes
- If the leader is in the majority group, it continues to operate normally. Followers in minority groups will timeout and become candidates, but they won't be able to gather enough votes to become leaders. So they will hang in candidate state until they reconnect.
- If the leader is in the minority group, it will eventually lose contact with the majority of nodes. Followers in the majority group will timeout and start a new election. A new leader will be elected in the majority group, while the old leader and its followers will become candidates but won't be able to gather enough votes. and will hang in candidate state until they reconnect.

So basically network partitions are handled by majority voting. Only the group with majority can elect a leader and make progress. Minority groups will be stuck until they reconnect.

### Reconnection of nodes
So in previous section i explained on what happens when nodes become unreachable. Now what happens when they reconnect?
It might be possible that during the partition, old leader caused his component to have dirty log writes not accepted by majority. So when nodes reconnect, we have to ensure that the logs are consistent.
- When a follower reconnects to the leader, it deletes all dirty writes (so uncommited)
- sends its current term and log index.
- The leader compares the follower's log with its own.
- The leader sends the necessary log entries to the follower to bring it up to date.
- The follower appends the received log entries and updates its term if necessary.

## 2. Concurrent elections
Ok so lets say we have nodes: A, B, C ... And A and B become candidates at the same time. What happens next?

### Examples
#### Example 1 (leader elected pretty casual case)
- we have 3 nodes: A, B, C
- Initial state: $F_{1}^{1}$, $F_{2}^{1}$, $F_{3}^{1}$
- A becomes candidate: $C_{1}^{2}$, $F_{2}^{1}$, $F_{3}^{1}$
- B becomes candidate: $C_{1}^{2}$, $C_{2}^{2}$, $F_{3}^{1}$
- C receives `RequestVote` from A: votes for A
- C receives `RequestVote` from B: sees term 2 (same as A), denies vote
- Resulting state: $C_{1}^{2}$, $C_{2}^{2}$, $F_{3}^{2}$ (A becomes leader)
> If C had received B's request first, it would have voted for B and denied A's request.

#### Example 2 (no leader elected edge case)
- we have 4 nodes: A, B, C, D
- Initial state: $F_1^1$, $F_2^1$, $F_3^1$, F_4^1$
- A becomes candidate: $C_1^2$, $F_2^1$, $F_3^1$, $F_4^1$
- B becomes candidate: $C_1^2$, $C_2^2$, $F_3^1$, $F_4^1$
- C receives `RequestVote` from A: votes for A
- C receives `RequestVote` from B: sees term 2 (same as A), denies vote
- D receives `RequestVote` from B: votes for B
- D receives `RequestVote` from A: sees term 2 (same as B), denies vote
- Resulting state: $C_1^2$, $C_2^2$, $F_3^2$, $F_4^2$ (no leader elected)
> In this case, neither A nor B received a majority of votes, so a new election will be started after a timeout, called a grace period. Each candidate will choose a random timeout within this grace period to avoid repeated collisions.

#### Example 3 (candidate but leader elected)
- we have 5 nodes: A, B, C
- Initial state: $L_1^1$, $F_2^1$, $F_3^1$
- A is a leader: $L_1^1$, $F_2^1$, $F_3^1$
- B becomes candidate: $L_1^1$, $C_2^2$, $F_3^1$
- C receives `RequestVote` from B: sees term 2 (higher than A), votes for B
- A receives `RequestVote` from B: sees term 2 (higher than its own), steps down to follower: $F_1^2$, $C_2^2$, $F_3^1$
- A sends heartbeat as follower: C sees term 2 (same as B), denies vote
- Resulting state: $F_1^2$, $C_2^2$, $F_3^2$ (B becomes leader)
> In this case, even though A was the leader initially, B's higher term caused A to step down (not sure if its correct).


# Two phase commit - idea for log replication
When a leader wants to replicate a log entry to followers, it uses a two-phase commit protocol to ensure consistency.
"Modification is commited to storage only after majority of nodes append it to their logs". Basically.
