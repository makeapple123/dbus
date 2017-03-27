package zk

import (
	"github.com/funkygao/dbus/pkg/cluster"
	log "github.com/funkygao/log4go"
	"github.com/funkygao/zkclient"
)

var (
	_ zkclient.ZkStateListener = &controller{}
)

type controller struct {
	kb *keyBuilder
	zc *zkclient.Client

	participantID string // in the form of host:port
	weight        int

	leaderID string

	pcl zkclient.ZkChildListener
	rcl zkclient.ZkChildListener
	lcl zkclient.ZkDataListener

	// only when participant is leader will this callback be triggered.
	onRebalance func(decision map[string][]string)
}

// New creates a Controller with zookeeper as underlying storage.
func New(zkSvr string, participantID string, weight int, onRebalance func(decision map[string][]string)) cluster.Controller {
	if onRebalance == nil {
		panic("onRebalance nil not allowed")
	}
	if len(zkSvr) == 0 {
		panic("invalid zkSvr")
	}
	if err := validateParticipantID(participantID); err != nil {
		panic(err)
	}

	return &controller{
		kb:            newKeyBuilder(),
		participantID: participantID,
		weight:        weight,
		onRebalance:   onRebalance,
		zc:            zkclient.New(zkSvr, zkclient.WithWrapErrorWithPath()),
	}
}

func (c *controller) connectToZookeeper() (err error) {
	log.Debug("connecting to zookeeper...")
	c.zc.SubscribeStateChanges(c)

	if err = c.zc.Connect(); err != nil {
		return
	}

	for retries := 0; retries < 3; retries++ {
		if err = c.zc.WaitUntilConnected(c.zc.SessionTimeout()); err == nil {
			log.Debug("connected to zookeeper")
			break
		}

		log.Warn("retry=%d %v", retries, err)
	}

	return
}

func (c *controller) RegisterResources(resources []string) error {
	for _, resource := range resources {
		path := c.kb.resource(resource)
		if err := c.zc.CreateEmptyPersistentIfNotPresent(path); err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) Start() (err error) {
	c.lcl = newLeaderChangeListener(c)
	c.pcl = newParticipantChangeListener(c)
	c.rcl = newResourceChangeListener(c)

	if err = c.connectToZookeeper(); err != nil {
		return
	}

	for _, path := range c.kb.persistentKeys() {
		if err = c.zc.CreateEmptyPersistentIfNotPresent(path); err != nil {
			return
		}
	}

	c.tryElect()
	return
}

func (c *controller) Close() (err error) {
	c.zc.Delete(c.kb.participant(c.participantID))
	c.zc.Disconnect()
	log.Trace("[%s] controller disconnected", c.participantID)
	return
}

func (c *controller) IsLeader() bool {
	// TODO refresh leader id?
	return c.leaderID == c.participantID
}
