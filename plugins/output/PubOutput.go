package output

import (
	"github.com/funkygao/dbus/engine"
	"github.com/funkygao/dbus/model"
	"github.com/funkygao/dbus/plugins/input/myslave"
	"github.com/funkygao/gafka/cmd/kateway/hh"
	"github.com/funkygao/gafka/cmd/kateway/hh/disk"
	"github.com/funkygao/gafka/cmd/kateway/meta"
	"github.com/funkygao/gafka/cmd/kateway/meta/zkmeta"
	"github.com/funkygao/gafka/cmd/kateway/store"
	"github.com/funkygao/gafka/cmd/kateway/store/kafka"
	"github.com/funkygao/gafka/ctx"
	"github.com/funkygao/gafka/zk"
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
)

type PubOutput struct {
	zone, cluster, topic string
	hhdirs               []string
	zkzone               *zk.ZkZone

	myslave *myslave.MySlave
}

func (this *PubOutput) Init(config *conf.Conf) {
	this.zone = config.String("zone", "")
	this.cluster = config.String("cluster", "")
	this.topic = config.String("topic", "")
	this.hhdirs = config.StringList("hhdirs", nil)
	if this.cluster == "" || this.zone == "" || this.topic == "" || len(this.hhdirs) == 0 {
		panic("invalid configuration")
	}

	this.zkzone = zk.NewZkZone(zk.DefaultConfig(this.zone, ctx.ZoneZkAddrs(this.zone)))

	meta.Default = zkmeta.New(zkmeta.DefaultConfig(), this.zkzone)
	meta.Default.Start()

	cfg := disk.DefaultConfig()
	cfg.Dirs = this.hhdirs
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	hh.Default = disk.New(cfg)
	if err := hh.Default.Start(); err != nil {
		panic(err)
	}

	store.DefaultPubStore = kafka.NewPubStore(100, 0, false, false, false)
	if err := store.DefaultPubStore.Start(); err != nil {
		panic(err)
	}
}

func (this *PubOutput) Run(r engine.OutputRunner, h engine.PluginHelper) error {
	this.myslave = engine.Globals().Registered("myslave").(*myslave.MySlave)

	for {
		select {
		case pack, ok := <-r.InChan():
			if !ok {
				return nil
			}

			row, ok := pack.Payload.(*model.RowsEvent)
			if !ok {
				log.Error("bad payload: %+v", pack.Payload)
				continue
			}

			msg, _ := row.Encode()
			partition, offset, err := store.DefaultPubStore.SyncPub(this.cluster, this.topic, nil, msg)
			if err != nil {
				log.Error("%s.%s.%s {%s} %v", this.zone, this.cluster, this.topic, row, err)
				hh.Default.Append(this.cluster, this.topic, nil, msg)
			}

			// FIXME only after pub'ed shall we mark it processed
			if err = this.myslave.MarkAsProcessed(row); err != nil {
				// TODO
			}

			log.Debug("%d/%d %s", partition, offset, row)

			pack.Recycle()
		}
	}

	return nil
}

func init() {
	engine.RegisterPlugin("PubOutput", func() engine.Plugin {
		return new(PubOutput)
	})
}
