package input

import (
	"github.com/funkygao/dbus/engine"
	"github.com/funkygao/dbus/pkg/model"
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
)

var (
	_ engine.Input = &MockInput{}
)

type MockInput struct {
	stopChan chan struct{}
}

func (this *MockInput) Init(config *conf.Conf) {
	this.stopChan = make(chan struct{})
}

func (this *MockInput) Stop(r engine.InputRunner) {
	log.Trace("[%s] stopping...", r.Name())
	close(this.stopChan)
}

func (this *MockInput) Run(r engine.InputRunner, h engine.PluginHelper) error {
	payload := model.Bytes("hello world")
	for {
		select {
		case <-this.stopChan:
			return nil

		case pack, ok := <-r.InChan():
			if !ok {
				log.Trace("yes sir!")
				break
			}

			pack.Payload = payload
			r.Inject(pack)
		}
	}

	return nil
}

func init() {
	engine.RegisterPlugin("MockInput", func() engine.Plugin {
		return new(MockInput)
	})
}
