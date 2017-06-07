# TODO

### TODO

- [ ] inc replication recv buffer size
- [ ] tweak of batcher yield
- [ ] participant starts slow
  - [06/06/17 15:06:11 CST] [TRAC] (     engine.go:281) engine starting...
  - [06/06/17 15:06:11 CST] [TRAC] (     engine.go:343) [10.9.1.1:9877] participant starting...
  - [06/06/17 15:06:41 CST] [INFO] (     engine.go:349) [10.9.1.1:9877] participant started
- [ ] debug-ability
- [ ] alert lags
- [ ] cluster
  - [ ] monitor resources cost and rebalance
  - [ ] support multiple projects
- [ ] resource group
- [ ] FIXME access denied leads to orphan resource
- [ ] myslave should have no checkpoint, placed in Input
- [ ] enhance Decision.Equals to avoid thundering herd
- [ ] myslave server_id uniq across the cluster
- [ ] add Operator for Filter
  - count, filter, regex, sort, split, rename
- [ ] RowsEvent avro
- [X] model.RowsEvent add dbus timestamp
- [X] HY000 auto heal
- [X] multiversion config in zk
- [X] model.RowsEvent add dbus timestamp
- [ ] controller
  - [ ] a participant is electing, then shutdown took a long time(blocked by CreateLiveNode)
  - [X] 2 phase rebalance: close participants then notify new resources
  - [X] what if RPC fails
  - [X] leader.onBecomingLeader is parallal: should be sequential
  - [ ] hot reload raises cluster herd: participant changes too much
  - [X] when leader make decision, it persists to zk before RPC for leader failover
  - [X] owner of resource
  - [X] leader RPC has epoch info
  - [ ] if Ack fails(zk crash), resort to local disk(load on startup)
  - [X] engine shutdown, controller still send rpc
  - test cases
    - [X] sharded resources
    - [X] brain split
    - [X] zk dies or kill -9, use cache to continue work
    - [X] kill -9 participant/leader, and reschedule
    - [X] cluster chaos monkey
- [ ] kafka producer qos
- [ ] batcher only retries after full batch ack, add timer?
- [ ] KafkaConsumer might not be able to Stop
- [ ] pack.Payload reuse memory, json.NewEncoder(os.Stdout)
- [X] kguard integration
- [X] router finding matcher is slow
- [X] hot reload on config file changed
- [X] each Input have its own recycle chan, one block will not block others
- [X] when Input stops, Output might still need its OnAck
- [X] KafkaInput plugin
- [X] use scheme to distinguish type of DSN
- [X] plugins Run has no way of panic
- [X] (replication.go:117) [zabbix] invalid table id 2968, no correspond table map event
- [X] make canal, high cpu usage
  - because CAS backoff 1us, cpu busy
- [X] ugly design of Input/Output ack mechanism
  - we might learn from storm bolt ack
- [X] some goroutine leakage
- [X] telemetry mysql.binlog.lag/tps tag name should be input name
- [X] pipeline
  - 1 input, multiple output
  - filter to dispatch dbs of a single binlog to different output
- [X] kill Packet.input field
- [X] visualized flow throughput like nifi
  - dump uses dag pkg
  - ![pipeline](https://github.com/funkygao/dbus-extra/blob/master/assets/dag.png?raw=true)
- [X] router metrics
- [X] dbusd api server
- [X] logging
- [X] share zkzone instance
- [X] presence and standby mode
- [X] graceful shutdown
- [X] master must drain before leave cluster
- [X] KafkaOutput metrics
  -  binlog tps
  -  kafka tps
  -  lag
- [X] hub is shared, what if a plugin blocks others
  - currently, I have no idea how to solve this issue
- [X] Batcher padding
- [X] shutdown kafka
- [X] zk checkpoint vs kafka checkpoint
- [X] kafka follower stops replication
- [X] can a mysql instance with miltiple databases have multiple Log/Position?
- [X] kafka sync produce in batch
- [X] DDL binlog
  - drop table y;
- [X] trace async producer Successes channel and mark as processed
- [X] metrics
- [X] telemetry and alert
- [X] what if replication conn broken
- [X] position will be stored in zk
- [X] play with binlog_row_image
- [ ] project feature for multi-tenant
- [ ] bug fix
  - [X] kill dbusd, dbusd-slave did not leave cluster
  - [X] next log position leads to failure after resume
  - [ ] KafkaOutput only support 1 partition topic for MysqlbinlogInput
  - [X] table id issue
  - [X] what if invalid position
  - [X] router stat wrong
    Total:142,535,625      0.00B speed:22,671/s      0.00B/s max: 0.00B/0.00B
  - [X] ffjson marshalled bytes has NL before the ending bracket
- [ ] test cases
  - [X] restart mysql master
  - [X] mysql kill process
  - [X] race detection
  - [ ] tc drop network packets and high latency
  - [ ] mysql binlog zk session expire
  - [X] reset binlog pos, and check kafka did not recv dup events
  - [X] MysqlbinlogInput max_event_length
  - [X] min.insync.replicas=2, shutdown 1 kafka broker then start
- [ ] GTID
  - place config to central zk znode and watch changes
- [ ] Known issues
  - Binlog Dump thread not close https://github.com/github/gh-ost/issues/292
- [ ] Roadmap
  - pubsub audit reporter
  - universal kafka listener and outputer

### Issues

- a big DELETE statement might kill dbusd
  - It might exceed max event size: 1MB
  - It might malloc a very big memory in RowsEvent struct
- OSC tools will make 'ALTER' very complex, whence dbusd not able to clear table columns cache