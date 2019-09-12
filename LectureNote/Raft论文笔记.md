# Raft论文笔记

[TOC]

Raft原版英文论文https://raft.github.io/raft.pdf

论文名称是寻找一种易于理解的一致性算法(扩展版)。Paxos一致性算法在解决分布式系统一致性方面一直是业界的标杆，但是由于其复杂性，导致Paxos难以理解，并且Paxos自身的算法结构需要进行大量的修改才能应用到实际的系统中。因此Raft一致性算法就应运而生。Raft算法通过将一致性问题分解为多个子问题(Leader election、Log Replication、Safety、Log Compaction、Membership change)来提升算法的可理解性。因此我先会对Raft算法整体运行机制进行简单综述，然后针对性的对每个子问题写一下我自己的理解，如果有不对的地方希望各位看官及时指出。

## 一	Raft算法综述

### 一致性问题

一致性问题在分布式存储系统中一直是个难点，也是重点。分布式存储系统为了满足可用性(Availability)，必须通过维护多个副本进行容错。当分布式系统中存在多个副本时，这些副本的一致性问题就又成了一个焦点问题。

一致性是分布式领域最为基础也是最重要的问题。如果分布式系统能实现一致，对外就可以呈现出一个完美的、可扩展的”虚拟节点“，相对物理节点具备更优越的性能和稳定性。在分布式系统中，运行着多个相互关联的服务节点。一致性是指分布式系统中的多个服务节点，给定一系列的操作，在约定协议的保障下，使它们对外界呈现的状态是一致的。在一个具有一致性的性质的集群里面，同一时刻所有的结点对存储在其中的某个值都有相同的结果，即对其共享的存储保持一致。

### 架构

![1](C:\Users\JKerving\Documents\跑路笔记\Raft\Raft1.png)

Raft算法从多副本状态机角度出发，用于管理多副本状态机的日志复制。图中可以看出replicated log会最终应用在replicated status machine中。

复制状态机通常都是基于复制日志实现的，如上图所示。每一个服务器存储一个包含一系列指令的日志，并且按照日志的顺序执行。每一个日志都按照相同的顺序包含相同的指令，所以每一个服务器都执行相同的指令序列。因为每个状态机都是确定的，每一次执行操作都产生相同的状态和同样的序列。因此只要保证所有服务器上的日志都完全一致，就可以保证state machine的一致性。日志的一致性是由一致性模块来保证的。

![](C:\Users\JKerving\Documents\跑路笔记\Raft\Figure1.png)

Figure1图中从client向server发出请求的交互过程说明一致性模块在replicated state machine中的作用。

1. 客户端向服务端发起请求，执行指定操作
2. Consensus Module将操作指令以日志的形式备份到其他备份实例上
3. 当日志中的log entry按照统一的顺序成功备份到各个实例上，日志中的log entry将会被应用到上层状态机
4. 服务端返回操作结果至客户端

整个复制状态机架构中最重要的就是一致性模块，保证复制日志相同就是一致性算法的工作。在一台服务器上，一致性模块接收客户端发送来的指令然后增加到自己的日志中去。它和其他服务器上的一致性进行通信来保证每一个服务器上的日志最终都以相同的顺序包含相同的请求，尽管有些服务器会宕机，但是要保证服务可用的话，宕机的服务器必须少于整体集群主机数的一半。一旦指令被正确的复制，每一个服务器的状态机按照日志顺序处理log entry，然后输出结果被返回给客户端。因此，服务器集群对外界呈现的是一个高可靠的状态机。



## 二	Raft基础

一个Raft集群包含若干个服务器节点；如果节点数是5个，这允许整个系统容忍2个节点的失效。在任何时刻，每一个服务器节点都处于这三个状态之一：领导人(Leader)、跟随者(Follower)、候选人(Candidate).

- Leader：所有请求的处理者，其实就是整个Raft集群和外部进行沟通的接口人。Leader接收client的更新请求，本地处理后再同步至多个其他副本；
- Follower：请求的被动更新者，从Leader接收更新请求，然后写入本地日志文件；
- Candidate：如果Follower副本在一段时间内没有收到Leader的心跳，则判断Leader可能已经发生故障，此时启动Leader election，Follower副本会变成Candidate状态，直至选主结束。

![](C:\Users\JKerving\Documents\跑路笔记\Raft\Figure4.png)

Figure4图中表示服务器在Follower、Candidate、Leader三个状态之间的转换。如果Follower在一定时间内没有收到来自Leader的消息，会转换为Candidate状态并触发election。获得集群中大多数节点选票的Candidate将转换为Leader。在一个任期内，Leader一直都会保持Leader状态除非自己宕机了。

![](C:\Users\JKerving\Documents\跑路笔记\Raft\Figure5.png)

Figure5：时间被划分成一个个的任期，每个任期开始都是一次选举。在选举成功后， 领导人会管理整个集群直到任期结束。有时候选举会失败，那么这个任期就会以没有领导人而结束。任期之间的切换可以在不同的服务器上观察到。

Raft把时间分割成任意长度的任期(term)，任期的概念我们可以理解为“皇帝的年号”。每个当选的领导人都有自己的任期。term有唯一的id。每一段任期从一次选举开始，一个或者多个Candidate尝试成为领导者。如果一个Candidate最终赢得选举，然后他就在接下来的term内充当领导人的职责。在某些情况下，一次选举过程会造成选票的瓜分。在这种情况下，这一term会以没有领导人结束；一个新的term会以一次新的选举而重新开始。Raft保证了在一个给定的任期内，最多只有一个领导者。

不同的server可能多次观察到任期之间的转换，但在某些情况下，一个节点也可能观察不到任何一次选举或者整个任期全程(比如节点宕机了)。任期在Raft算法中充当logic clock，这允许server可以查明一些过期的信息比如陈旧的领导者。每一个节点存储一个当前的任期号(current_term_id)，term_id在整个时期内是单调增长的。当服务器之间通信的时候会交换current_term_id；如果一个server的current_term_id比其他server小，那么它会更新自己的current_term_id到较大的编号值。如果一个Candidate或者Leader发现自己的current_term_id过期了，那么他会立即恢复成Follower状态。如果一个节点接收到一个包含过期的current_term_id的请求，那么他会直接拒绝这个请求。

Raft算法中server节点之间通信使用远程过程调用(RPCs),并且基本的一致性算法只需要两种类型的RPCs。请求投票(RequestVote) RPCs由候选人在选举期间发起，然后附件条目(AppendEntries) RPCs由领导人发起，用来复制日志和提供一种心跳机制。后面为了在server之间传输snapshot增加了第三种RPC。当服务器没有及时的收到RPC响应，会进行重试，并且他们能够并行的发起RPCs来获得更高的性能。

## 三	领导人选举

Raft使用一种心跳机制来触发领导人选举。当服务器程序启动时，他们都是Follower角色。一个server节点继续保持着Follower状态只要它从Leader或者Candidate处接收到有效的RPCs。Leader周期性的向所有Follower发送心跳包(其实就是不包含日志项内容的AppendEntries RPCs)来维持自己的领导权威。如果一个Follower在一段时间内没有接收到任何消息，也就是发生了timeout，那么他就会认为系统中没有Leader，因此自己就会发起选举以选出新的Leader。

每一段term一开始的选举过程：

1. Follower将自己维护的current_term_id+1；
2. 然后转换状态为Candidate；
3. 发送RequestVoteRPC消息(会带上自己的current_term_id)给其他所有server；

要开始一次选举过程，Follower先要增加自己的current_term_id并转换到Candidate状态。然后它会并行的向集群中的其他服务器节点发送RequestVoteRPC来给自己投票。Candidate会继续保持当前状态直到以下三种情况出现：

1. 自己已经赢得了选举，成功被选举为Leader。当收到了大多数节点(majority)的选票后，角色状态会转换为Leader，之后会定期给其它所有server发心跳信息(不带log entry的AppendEntries RPC)，用来告诉对方自己是当前term(也就是发送RequestVoteRPC时附带的current_term_id)的Leader。每个term最多只有一个leader，term id作为logical clock，在每个RPC消息中都会带上，用于检测过期的消息。当一个server收到的RPC消息中的rpc_term_id比本地的current_term_id更大时，就更新current_term_id为rpc_term_id，并且如果当前节点的角色状态为leader或者candidate时，也会将自己的状态切换为follower。如果rpc_term_id比接收节点本地的current_term_id更小，那么RPC消息就被会拒绝。
2. 其他节点最终成功被选举为Leader。当Candidate在等待投票的过程中，收到了rpc_term_id大于或者等于本地的current_term_id的AppendEntriesRPC消息时，并且这个RPC消息声明自己是这个任期内的leader。那么收到消息的节点将自己的角色状态转换为follower，并且更新本地的current_term_id。
3. 第三种可能的结果是Candidate既没有赢得选举也没有输，本轮选举没有选出leader，这说明投票被瓜分了。没有任何一个Candidate收到了majority的投票时，leader就无法被选出。这种情况下，每个Candidate等待的投票的过程就出现timeout，随后candidates都会将本地的current_term_id+1，再次发起RequestVoteRPC进行新一轮的leader election。

- 每个节点只会给每个term投一票
- Raft算法使用随机选举超时时间的方法来确保很少会发生选票被瓜分的情况，就算发生也能很快的解决。为了阻止选票起初就被瓜分，选举超时时间是从一个固定的区间(150-300ms)随机选择。这样可以把服务器都分散开以至于在大多数情况下只有一个服务器会选举超时；然后他赢得选举并在其他服务器超时之前发送心跳包。同样的机制也用在了选票被瓜分的情况下。当选票被瓜分，所有candidate同时超时，有很大可能又进入新一轮的选票被瓜分循环中。为了避免这个问题，每个candidate的选举超时时间从150-300ms中随机选取，那么第一个超时的candidate就可以率先发起新一轮的leader election，带着最大的current_term_id给其他所有server发送RequestVoteRPC消息，从而自己成为leader，然后给他们发送AppendEntriesRPC以告诉他们自己是leader。



## 四	日志复制

一旦一个leader被选举出来，他就开始为客户端提供服务，接受客户端发来的请求。每个请求包含一条需要被replicated state machine执行的指令。leader会把每条指令作为一个最新的log entry添加到日志中，然后并发的向其他服务器发起AppendEntriesRPC请求，让他们也复制这条指令。当leader确认这条log entry被安全地复制(大多数副本已经改日志指令写入本地日志中)，leader就会将这条log entry应用到状态机中然后返回结果给客户端。如果follower崩溃或者运行缓慢，没有成功的复制日志，leader会不断的重复尝试AppendEntriesRPCs(尽管已经将执行结果返回给客户端)直到所有的follower都最终存储了所有的log entry。

![](C:\Users\JKerving\Documents\跑路笔记\Raft\Figure6.png)

Figure6：日志由有序序号标记的条目组成。每个条目都包含创建时的任期号和一个状态机需要执行的指令。一个条目当可以安全的被应用到状态机中去的时候，就认为是可以提交了。

日志的基本组织结构：

- 每一条日志都有日志序号log index
- 每一条日志条目包含状态机要执行的日志指令(x←3)、该日志指令对应的term

leader来决定什么时候把log entry应用到状态机中是安全的；这种log entry被称为已提交。Raft算法保证所有commited的log entry都是持久化的并且最终会被所有可用的状态机执行。在leader创建的log entry复制到大多数的服务器节点的时候，log entry就会被提交。同时，leader的日志中之前的所有log entry也都会被提交，包括由其他leader创建的条目。一旦follower知道一条log entry已被提交，那么这个节点也会将这个 log entry按照日志的顺序应用到本地的状态机中。

Raft算法日志机制有以下2个特性：

- 如果在不同的日志中的两个log entry拥有相同的index和term_id，那么他们存储了相同的指令
- 如果在不同的日志中的两个log entry拥有相同的index和term_id，那么他们之前的所有log entry也全部相同

第一个特性是因为leader最多在一个任期里在指定的一个日志索引位置创建一个log entry，同时log entry在日志中的位置也从来不会改变。

第二个特性是通过AppendEntriesRPC的一个简单的一致性检查保证。在发送AppendEntriesRPC时，leader会把新的log entry紧接着之前的log entry的index和term_id一起包含在内。如果follower在它的日志中找不到包含相同index和term_id的log entry，那么他就会拒绝接收新的log entry。这种一致性检查会保证每一次新追加的log entry的一致性。一开始空的日志状态肯定满足日志匹配特性，然后在日志扩展时AppendEntriesRPC的一致性检查保护了这种特性。

在正常情况下，leader和follower的日志保持一致性。但是leader崩溃会使得日志处于不一致的状态。当一个新leader被选出来时，它的日志和其他的follower的日志可能不一样，这时就需要一个机制来保证日志的一致性。一个新leader产生时，集群状态可能会像下图一样：

![Figure7](C:\Users\JKerving\Documents\跑路笔记\Raft\Figure7.png)

当新leader成功当选时，follower可能是任何情况(a-f)。每个格子表示是一个log entry；里面的数字表示term_id。follower可能会缺少一些log entry(a-b)，可能会有一些未被提交的log entry(c-d)，或者两种情况都存在(e-f)。

简单解释一下场景f出现的情况：某个服务器节点在任期2的时候是leader，

因此需要一种机制来让leader和follower对log达成一致。leader会为每个follower维护一个nextIndex，表示leader给各个follower发送的下一条log entry在log中的index，初始化为leader的最后一条log entry的下一个位置。leader给follower发送AppendEntriesRPC消息，带着(term_id,(nextIndex-1))，term_id是索引位置为(nextIndex-1)的log entry的term_id，follower接收到AppendEntriesRPC后，会从自己的log的对应位置找是否有log entry能够完全匹配上。如果不存在，就给leader回复拒绝消息，然后leader再将nextIndex-1，再重复，直至AppendEntriesRPC消息被follower接收，也就是leader和follower的log entry能够匹配。

以leader和f为例：

leader的最后一条log entry的index是10，因此初始化时，nextIndex=11，leader发f发送AppendEntriesRPC(6,10)，f在节点本地日志的index=10的位置上没有找到term_id=6的log entry。则给leader回应一个拒绝消息。随后leader将nextIndex-1，变为10，然后给f发送AppendEntriesRPC(6,9)，f在自己的log的index=9的位置没有找到term_id=6的log entry。匹配过程会一直循环下去直到leader和follower的日志能够匹配。当leader发送了AppendEntriesRPC(1,3)，f在自己log的index=3的位置找到了term_id为1的log entry。成功接收leader的消息。随后，leader就开始从index=4的位置开始给f推送日志。

## 五	安全性

前面写的内容描述了Raft算法是如何选举和复制日志的。然而，到目前为止描述的机制并不能充分的保证每一个状态机会按照相同的顺序执行相同的指令。比如一个follower可能会进入不可用状态时领导人已经提交了多条log entry，随后这个follower恢复后可能会被选举为leader并覆盖这些log entry。因此导致了不同的状态机可能会执行不同的日志指令。

因此这节要讨论的就是**哪些follower有资格成为leader**

Raft保证被选为新leader的节点拥有所有committed的log entry，这与ViewStamped Replication不同，后者不需要这个保证，而是通过其他机制从follower拉取自己没有提交的日志记录。

这个保证是在RequestVoteRPC阶段实现的，candidate在发送RequestVoteRPC时，会带上自己的最后一条log entry的term_id和index，其他节点收到消息时，如果发现自己的本地日志比RPC请求中携带的更新，则拒绝投票。日志比较的原则是，如果本地的最后一条log entry的term_id更大，则更新。如果term_id相同，则日志条目更多的一方更新(index大的一方日志条目最多)。

![](C:\Users\JKerving\Documents\跑路笔记\Raft\Figure8.png)

1. 在阶段a，term=2，S1是leader，且S1写入日志(term,index)为(2,2),日志也被同步写入了S2；
2. 在阶段b，S1离线，触发一次新的选主，此时S5被选为新的Leader，此时term=3，且写入了日志(term,index)为(3,2)；
3. S5尚未将日志推送到Followers变离线了，从而又触发了一次新的选主，而之前离线的S1经过重新上线后被选中为leader，此时系统term=4。随后S1会将自己的日志同步到Followers，图c就是将日志(2,2)同步到S3，而此时由于该log entry已经被同步到了多数节点(S1,S2,S3)，因此log entry(2,2)可以被commit；
4. 在阶段d，S1又变为离线，系统触发一次选主，而S5有可能被选为新的leader。S5满足竞选成为leader的条件：1.S5的最后一个log entry的term=3，多数节点的最后一条log entry的term=2；2.最新的日志index=2，比大多数节点的日志都新。因此当S5成功被选为新的leader后，会将自己的日志更新到Followers，于是S2、S3中已经被提交的日志(2,2)被覆盖了。然而一致性协议中不允许出现已经apply到state machine中的日志被覆盖。

因此为了避免发生这种错误，需要对协议进行微调：

> 只允许Leader提交当前term的日志

经过微调后，即使日志(2,2)已经被多数节点(S1、S2、S3)确认了，但是不能被commit，因为当前term=4，而(2,2)是来自之前term=2的日志，直到S1在当前term=4产生的日志(4,3)被大多数Followers确认，S1才能够commit(4,3)这条日志。而根据Raft机制，leader复制本地日志到各个Followers时，会通过AppendEntriesRPC进行一致性检查。(4,3)之前的所有日志也会被commit。此时即使S1再下线，重新选主时S5也不可能选主成为leader，因为它没有包含大多数节点已经拥有的日志(4,3).

- 什么时候一条log entry可以被认为是commited？

leader正在replicate当前term的日志记录给follower，一旦leader确认了这条log entry被majority写盘了，这条log entry就被认为commited。如Figure8中的图a，S1作为term2阶段的leader，如果index=2的log entry被majority写盘了，这条log entry就被认为是commited

leader正在replicate更早的term的log entry给其它follower，如图c。S1是term4阶段的leader，正在将term=2，index=2的log entry复制给其它follower。

### Follower和Candidate崩溃







