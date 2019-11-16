# 分布式系统基础理论(二) CAP定理

CAP定理是分布式系统，特别是分布式存储领域中被讨论最多的理论。CAP是由Eric Brewer在2000年PODC会议上提出，是Eric Brewer在Inktomi期间研发搜索引擎、分布式web缓存时得出的关于数据一致性(consistency)、服务可用性(availability)、分区容错性(partition-tolerance)的猜想：

> It is impossible for a web service to provide the three following guarantees:Consistency,Availability and Patition-tolerance.

CAP理论是在“数据一致性和可用性”的争论中产生的。Brewer在90年代就开始研究基于集群的跨区域系统，这种类型的系统更加重视Availability，因此他们采用了缓存或者事后更新的方式来优化系统的可用性。但是这种数据更新方式就会牺牲系统数据一致性。

Brewer提出的猜想在两年后被证明成立，称为现在业界熟知的**CAP定理**：

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191115171825.png)

- **数据一致性(Consistency)**:

  > Any read operation that begins after a write operation completes must return that value,or the result of a later write operation

  在一个一致性的系统中，客户端向任何服务器发起一个写请求，将一个值写入服务器并成功得到响应，那么之后向这个分布式系统内的任何节点发起读请求，都必须读取到这个值，或者读到更近写操作中写入的值。如果返回失败，那么所有读操作都不能读到这个数据，对调用者而言数据具有强一致性(strong consistency)。强一致性也叫原子性atomic、线性一致性(linearizable consistency)

- **可用性(Availability)**:

  > every request received by a non-failing node in the system must result in a response
  >
  > 在一个可用的分布式系统中，客户端向其中一个服务器节点发起一个请求且该服务器未崩溃，那么这个服务器最终必须响应客户端的请求，但是不能保证返回的是最新写入的数据

- **分区容错性(Partition tolerance)**:

  > the nerwork will be allowed to lose arbitrarily many messages sent from one node to another
  >
  > 尽管任意数量的消息在节点间传输过程中丢失(网络传输过程中丢失或者长时间延迟)，我们的系统在任意的网络分区情况下仍能正常对外服务

在某时刻如果满足AP，分隔的节点同时对外服务但不能相互通信，将导致状态不一致，即不能满足C；如果满足CP，网络分区的情况下为达成C，请求只能一直等待，即不满足A；如果要满足CA，在一定时间内要达到节点状态一致，要求不能出现网络分区，则不能满足P。

CAP三者最多只能满足其中两个，和FLP定理一样，CAP定理也指示了一个不可达的结果。但是Brewer提出的三选二的思想也在一定程度上带来了很多误解，在工程实践中存在很多现实限制条件，需要我们做更多地考量与权衡，避免进入CAP认识误区。

我们通过一个具体的例子来对CAP定理进行说明。

让我们来考虑一个非常简单的分布式系统，它由两台服务器G1和G2组成，这两台机器都存储的相同的数据v，v的初始值为v0。G1和G2相互之间能够通信，并且也能与外部的客户端通信。具体分布式系统架构图如下所示：

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191115173731.png)

客户端可以向任何服务器发出读写请求。服务器当接收到请求后，将在一定时间内返回对请求的响应结果，然后把响应结果返回给客户端。如下图，client向G1发起写请求

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191115234305.png)

下图是一个读请求的例子，客户端发起读请求

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191115234644.png)

上面就是简单的分布式系统就建立起来了。


* [分布式系统基础理论(二) CAP定理](#分布式系统基础理论二-cap定理)
   * [分区容错性(Patition tolerance)](#分区容错性patition-tolerance)
   * [可用性(Availability)](#可用性availability)
   * [一致性(Consistency)](#一致性consistency)
      * [一致性模型](#一致性模型)
      * [数据不一致性示例](#数据不一致性示例)
      * [数据一致性示例](#数据一致性示例)
   * [CAP定理的证明](#cap定理的证明)
   * [CAP的新认知](#cap的新认知)
   * [跳出CAP](#跳出cap)
   * [小结](#小结)


## 分区容错性(Patition tolerance)

大多数分布式系统都分布在多个子网络，每个子网络就叫做一个区(partiton)。在实际生产系统中，正常情况下分布式系统各个节点之间的通信是可靠的，不会出现消息丢失或者延迟很高的情况。但是网络情况往往不尽如人意，总会出现消息丢失或者消息延迟很高的情况。这时候不同区域的节点在一段时间内就会出现无法通信的情况，也就发生了分区。

分区容忍是指分布式系统在出现网络分区时，仍然能对外提供服务，外部客户端对于网络分区现象是无感知的。这里面提到的对外提供服务与可用性的要求不一样，可用性要求的是对于任意请求都能够得到响应，意味着即使出现网络分区所有节点都能够提供服务。而分区容错性重点在于出现网络分区后，系统作为一个整体是可用的。

举例来说，G1和G2之间相互发送的任意消息都丢失了，那么系统就出现分区现象。

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191116003633.png)

## 可用性(Availability)

只要服务器节点收到用户的请求，服务器就必须给出回应。用户可以选择向G1或G2发起读操作，无论是哪台服务器，只要收到该请求，就必须返回结果给用户服务器中存储的值是多少，否则就不满足可用性。

## 一致性(Consistency)

一致性的意思是写操作之后的读操作，必须返回该值。“all nodes see the same data at the same time”，即更新操作成功并返回客户端完成后，所有节点在同一时间的数据完全一致，所以，一致性说的就是数据一致性。

对于一致性，可以分为从客户端和服务端两个不同的视角。从客户端来看，一致性主要指的是多并发访问时更新过的数据如何获取的问题。从服务端来看，则是更新如何复制数据分布到整个系统，以保证数据最终一致性。

一致性是因为有并发读写才有的问题，因此在理解一致性的问题时，一定要注意结合考虑并发读写的场景。

从客户端角度，多并发访问时，更新过的数据在不同进程如何获取的不同策略，决定了不同的一致性。

### 一致性模型

说起数据一致性来说，简单说有三种类型：

1. Weak弱一致性：当客户端写入一个新值后，读操作在数据副本上可能读出来，也可能读不出来。比如：某些cache系统，网络游戏其他玩家的数据和你没什么关系，VOIP这样的系统。
2. Eventually最终一致性：当客户端写入一个新值后，有可能读不出来，但在某个时间窗口之后保证最终能读出来的。比如：DNS，电子邮件、Amazon S3，Google搜索引擎。
3. Strong强一致性：新的数据一旦写入，在任意副本任意时刻都能读到新值。比如：文件系统、关系型数据库都是强一致性的。

从三种一致性的模型上来说，可以看到Weak和Eventually一般来说是异步冗余的，而Strong一般来说是同步冗余的，异步情况通常意味着更好的性能，但也意味着更复杂的状态控制。强一致性意味着简单，但是意味着性能下降。

讨论一致性的时候必须要明确一致性的范围，即在一定的边界内状态是一致的，超出边界之外的一致性是无从谈起的。比如Paxos在发生网络分区的时候，在一个主分区内可以保证完备的一致性和可用性，而在分区外服务是不可用的。当系统在分区的时候选择了一致性，也就是CP，并不意味着完全失去了可用性，这取决于一致性算法的实现。

### 数据不一致性示例

下面通过一个不一致的分布式系统的例子来说明：

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191116104630.png)

客户端向G1发起写请求，将v的值更新为v1且得到G1的确认响应；当向G2发起读v的请求时，读取到的却是旧的值v0，与期待的v1不一致。

### 数据一致性示例

下面是数据一致的分布式系统的例子：

从上面的不一致的分布式系统来看，如果想要为了数据一致性，需要让G2也能变为v1，就要在G1写操作的时候，让G1向G2发送一条消息，要求G2也改成v1。

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191116105706.png)

这样用户向G2发起读操作，也能得到v1。

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191116105739.png)

## CAP定理的证明

经过上面的讲解，大家可能会CAP定理有了一个基本的概念，我们可以来证明一个系统不能同时满足这三种属性。

假设存在一个同时满足这三个属性的系统，我们第一件要做的就是让系统发生网络分区，如下图情况一样：

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191116110110.png)

分布式系统发生了网络分区，G1和G2之间无法通信，被分隔为两个子区。

客户端向G1发起写请求，将v的值更新为v1，因为系统是可用的，所以G1必须响应客户端的请求，但是由于网络是分区，G1无法将其数据复制到G2.

![image-20191116110550833](/Users/jkerving/Library/Application Support/typora-user-images/image-20191116110550833.png)

由于网络分区导致不一致

接着，客户端向G2发起读v的请求，再一次因为系统是可用的，所以G2必须响应客户端的请求，又由于网络是分区的，G2无法从G1更新v的值，所以G2返回给客户端的是旧的值v0。

![](https://raw.githubusercontent.com/KyrieJK/Figurebed/master/img/20191116110826.png)

客户顿发起写请求将G1上v的值修改为v1之后，从G2上读取到的值仍然是v0，这违背了数据一致性。

## CAP的新认知

CAP经常被误解，很大程度上是因为在讨论CAP的时候可用性和一致性的作用范围往往都是模糊的。如果不先定义好可用性、一致性、分区容忍在具体场景下的概念，CAP实际上反而会束缚系统设计的思路。首先，由于分区很少发生，那么在系统不存在分区的情况下没什么理由牺牲C或A。其次，C与A之间取舍可以在同一系统内以非常细小的粒度反复发生，而每一次的决策可能因为具体的操作，乃至因为牵涉到特定的数据或用户有所不同。最后，这三种性质都可以在程度上都可以进行度量，并不是非黑即白的有或无。可用性显然是在0%到100%之间连续变化的，一致性分很多级别，甚至分区也可以细分为不同含义，如系统内的不同部分对于是否存在分区可以有不一样的认知。

## 跳出CAP

CAP理论对实现分布式系统具有指导意义，但CAP理论并没有涵盖分布式工程实践中的所有重要因素。

例如延时(latency)，它是衡量系统可用性、与用户体验直接相关的一项重要指标。CAP理论中的可用性要求操作能终止、不无休止地进行，除此之外，我们还关心到底需要多长时间能结束操作，这就是延时，它值得我们设计、实现分布式系统时单列出来考虑。

延时与数据一致性也是一对矛盾点，如果达到强一致性、多个副本数据一致，必然降低系统性能，增加延时。加上延时的考量，我们得到一个CAP理论的修改版本PACELC：如果出现P(网络分区)，如何在A(服务可用性)、C(数据一致性)之间选择；否则，如何在L(延时)、C(数据一致性)之间选择。

## 小结

以上介绍了CAP理论的源起和发展，介绍了CAP理论给分布式系统工程实践带来的启示。

CAP理论对分布式系统实现有非常重大的影响，我们可以根据自身的业务特点，在数据一致性和服务可用性之间作出倾向性地选择。通过放松约束条件，我们可以实现在不同时间点满足CAP，其中C可以由强一致性松弛为最终一致性。

推荐一些论文或者Blog给各位看一下：

[1] [Harvest, Yield, and Scalable Tolerant Systems](https://link.zhihu.com/?target=https%3A//cs.uwaterloo.ca/~brecht/servers/readings-new2/harvest-yield.pdf), Armando Fox , Eric Brewer, 1999

[2] [Towards Robust Distributed Systems](https://link.zhihu.com/?target=http%3A//www.cs.berkeley.edu/~brewer/cs262b-2004/PODC-keynote.pdf), Eric Brewer, 2000

[3] [Inktomi's wild ride - A personal view of the Internet bubble](https://link.zhihu.com/?target=https%3A//www.youtube.com/watch%3Fv%3DE91oEn1bnXM), Eric Brewer, 2004

[4] [Brewer’s Conjecture and the Feasibility of Consistent, Available, Partition-Tolerant Web](https://link.zhihu.com/?target=https%3A//pdfs.semanticscholar.org/24ce/ce61e2128780072bc58f90b8ba47f624bc27.pdf), Seth Gilbert, Nancy Lynch, 2002

[5] [Linearizability: A Correctness Condition for Concurrent Objects](https://link.zhihu.com/?target=http%3A//cs.brown.edu/~mph/HerlihyW90/p463-herlihy.pdf), Maurice P. Herlihy,Jeannette M. Wing, 1990

[6] [Brewer's CAP Theorem - The kool aid Amazon and Ebay have been drinking](https://link.zhihu.com/?target=http%3A//julianbrowne.com/article/viewer/brewers-cap-theorem), Julian Browne, 2009

[7] [CAP Theorem between Claims and Misunderstandings: What is to be Sacrificed?](https://link.zhihu.com/?target=http%3A//www.sersc.org/journals/IJAST/vol56/1.pdf), Balla Wade Diack,Samba Ndiaye,Yahya Slimani, 2013

[8] [Errors in Database Systems, Eventual Consistency, and the CAP Theorem](https://link.zhihu.com/?target=http%3A//cacm.acm.org/blogs/blog-cacm/83396-errors-in-database-systems-eventual-consistency-and-the-cap-theorem/fulltext), Michael Stonebraker, 2010

[9] [CAP Confusion: Problems with 'partition tolerance'](https://link.zhihu.com/?target=http%3A//blog.cloudera.com/blog/2010/04/cap-confusion-problems-with-partition-tolerance/), Henry Robinson, 2010

[10] [You Can’t Sacrifice Partition Tolerance](https://link.zhihu.com/?target=https%3A//codahale.com/you-cant-sacrifice-partition-tolerance/), Coda Hale, 2010

[11] [Perspectives on the CAP Theorem](https://link.zhihu.com/?target=https%3A//groups.csail.mit.edu/tds/papers/Gilbert/Brewer2.pdf), Seth Gilbert, Nancy Lynch, 2012

[12] [CAP Twelve Years Later: How the "Rules" Have Changed](https://link.zhihu.com/?target=https%3A//www.computer.org/cms/Computer.org/ComputingNow/homepage/2012/0512/T_CO2_CAP12YearsLater.pdf), Eric Brewer, 2012

[13] [How to Make a Multiprocessor Computer That Correctly Executes Multiprocess Programs](https://link.zhihu.com/?target=http%3A//research.microsoft.com/en-us/um/people/lamport/pubs/multi.pdf), Lamport Leslie, 1979

[14] [Eventual Consistent Databases: State of the Art](https://link.zhihu.com/?target=http%3A//www.ronpub.com/publications/OJDB-v1i1n03_Elbushra.pdf), Mawahib Elbushra , Jan Lindström, 2014

[15] [Eventually Consistent](https://link.zhihu.com/?target=http%3A//www.allthingsdistributed.com/2008/12/eventually_consistent.html), Werner Vogels, 2008

[16] [Speed Matters for Google Web Search](https://link.zhihu.com/?target=http%3A//www.isaacsunyer.com/wp-content/uploads/2009/09/test_velocidad_google.pdf), Jake Brutlag, 2009

[17] [Consistency Tradeoffs in Modern Distributed Database System Design](https://link.zhihu.com/?target=http%3A//cs-www.cs.yale.edu/homes/dna/papers/abadi-pacelc.pdf), Daniel J. Abadi, 2012

[18] [A CAP Solution (Proving Brewer Wrong)](https://link.zhihu.com/?target=http%3A//guysblogspot.blogspot.com/2008/09/cap-solution-proving-brewer-wrong.html), Guy's blog, 2008

[19] [How to beat the CAP theorem](https://link.zhihu.com/?target=http%3A//nathanmarz.com/blog/how-to-beat-the-cap-theorem.html), nathanmarz , 2011

[20] [The CAP FAQ](https://link.zhihu.com/?target=https%3A//github.com/henryr/cap-faq), Henry Robinson
