package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/funkygao/dbus/engine"
	"github.com/funkygao/dbus/model"
	"github.com/funkygao/dbus/plugins/input/myslave"
	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/gafka/zk"
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
)

type sentPos struct {
	Log string `json:"log"`
	Pos uint32 `json:"pos"`

	f *os.File
}

func (p *sentPos) load(fn string) {
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_RDWR|os.O_SYNC, 0600)
	if err != nil {
		panic(err)
	}

	p.f = f

	dec := json.NewDecoder(f)
	dec.Decode(p)
}

func (p *sentPos) dump() {
	p.f.Seek(0, os.SEEK_SET)
	enc := json.NewEncoder(p.f)
	enc.Encode(p)
}

func (p *sentPos) close() {
	p.dump()
	p.f.Close()
}

type KafkaOutput struct {
	zone, cluster, topic string

	zkzone   *zk.ZkZone
	ack      sarama.RequiredAcks
	async    bool
	compress bool

	p  sarama.SyncProducer
	ap sarama.AsyncProducer

	sendMessage func(row *model.RowsEvent)

	pos      *sentPos
	posDirty bool

	// FIXME should be shared with MysqlbinlogInput
	// currently, KafkaOutput MUST setup master_host/master_port to correctly checkpoint position
	myslave *myslave.MySlave
}

func (this *KafkaOutput) Init(config *conf.Conf) {
	this.zone = config.String("zone", "")
	this.cluster = config.String("cluster", "")
	this.topic = config.String("topic", "")
	if this.cluster == "" || this.zone == "" || this.topic == "" {
		panic("invalid configuration")
	}

	this.pos = &sentPos{}
	this.pos.load(fmt.Sprintf("_%s.%s.%s.dmp", this.zone, this.cluster, this.topic))
	log.Debug("[%s.%s.%s] %+v", this.zone, this.cluster, this.topic, this.pos)

	// ack is ignored in async mode
	this.ack = sarama.RequiredAcks(config.Int("ack", int(sarama.WaitForLocal)))
	this.async = config.Bool("async", false)
	if this.async {
		this.sendMessage = this.asyncSendMessage
	} else {
		this.sendMessage = this.syncSendMessage
	}
	this.compress = config.Bool("compress", false)
	this.zkzone = zk.NewZkZone(zk.DefaultConfig(this.zone, ctx.ZoneZkAddrs(this.zone)))
}

func (this *KafkaOutput) Run(r engine.OutputRunner, h engine.PluginHelper) error {
	this.myslave = engine.Globals().Registered("myslave").(*myslave.MySlave)

	if err := this.prepareProducer(); err != nil {
		return err
	}

	posTicker := time.NewTicker(time.Millisecond * 100)
	defer func() {
		posTicker.Stop()
		if this.async {
			this.ap.Close()
		} else {
			this.p.Close()
		}
	}()

	for {
		select {
		case <-posTicker.C:
			if this.posDirty {
				this.pos.dump()
				this.posDirty = false
			}

		case pack, ok := <-r.InChan():
			if !ok {
				this.pos.close()
				return nil
			}

			row, ok := pack.Payload.(*model.RowsEvent)
			if !ok {
				log.Error("bad payload: %+v", pack.Payload)
				continue
			}

			// best effort to reduce dup message to kafka
			if row.Log > this.pos.Log || (row.Log == this.pos.Log && row.Position > this.pos.Pos) {
				this.sendMessage(row)
			}

			pack.Recycle()
		}
	}

	return nil
}

func (this *KafkaOutput) prepareProducer() error {
	cf := sarama.NewConfig()
	cf.ChannelBufferSize = 256 * 2 // default was 256
	if this.compress {
		cf.Producer.Compression = sarama.CompressionSnappy
	}

	zkcluster := this.zkzone.NewCluster(this.cluster)

	if !this.async {
		cf.Producer.RequiredAcks = this.ack
		p, err := sarama.NewSyncProducer(zkcluster.BrokerList(), cf)
		if err != nil {
			return err
		}

		this.p = p
		return nil
	}

	// async producer
	cf.Producer.Return.Errors = true
	cf.Producer.Return.Successes = true
	cf.Producer.Retry.Backoff = time.Millisecond * 300
	cf.Producer.Retry.Max = 3
	cf.Producer.RequiredAcks = sarama.NoResponse
	cf.Producer.Flush.Frequency = time.Second
	cf.Producer.Flush.Messages = 2000 // TODO
	cf.Producer.Flush.MaxMessages = 0 // unlimited
	ap, err := sarama.NewAsyncProducer(zkcluster.BrokerList(), cf)
	if err != nil {
		return err
	}

	this.ap = ap
	go func() {
		for {
			select {
			case msg := <-this.ap.Successes():
				row := msg.Value.(*model.RowsEvent)
				this.markAsSent(row)
				if err := this.myslave.MarkAsProcessed(row); err != nil {
					log.Error("[%s.%s.%s] {%s} %v", this.zone, this.cluster, this.topic, row, err)
				}

			case err := <-this.ap.Errors():
				// e,g.
				// kafka: Failed to produce message to topic dbustest: kafka server: Message was too large, server rejected it to avoid allocation error.
				row := err.Msg.Value.(*model.RowsEvent)
				log.Error("[%s.%s.%s] %s %s", this.zone, this.cluster, this.topic, err, row.MetaInfo())
			}
		}

	}()

	return nil
}

func (this *KafkaOutput) syncSendMessage(row *model.RowsEvent) {
	msg := &sarama.ProducerMessage{
		Topic: this.topic,
		Value: row,
	}

	var (
		partition int32
		offset    int64
		err       error
	)
	for {
		if partition, offset, err = this.p.SendMessage(msg); err == nil {
			break
		}

		log.Error("%s.%s.%s {%s} %v", this.zone, this.cluster, this.topic, row, err)
		time.Sleep(time.Second)
	}

	this.markAsSent(row)
	if err = this.myslave.MarkAsProcessed(row); err != nil {
		log.Warn("%s.%s.%s {%s} %v", this.zone, this.cluster, this.topic, row, err)
	}

	log.Debug("sync sent [%d/%d] %s", partition, offset, row)
}

func (this *KafkaOutput) asyncSendMessage(row *model.RowsEvent) {
	log.Debug("async sending: %s", row)

	this.ap.Input() <- &sarama.ProducerMessage{
		Topic: this.topic,
		Value: row,
	}
}

func (this *KafkaOutput) markAsSent(row *model.RowsEvent) {
	this.pos.Log = row.Log
	this.pos.Pos = row.Position
	this.posDirty = true
}

func init() {
	engine.RegisterPlugin("KafkaOutput", func() engine.Plugin {
		return new(KafkaOutput)
	})
}
