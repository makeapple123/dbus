package engine

import (
	"fmt"
	"net/http"

	"github.com/funkygao/dbus"
	"github.com/funkygao/dbus/pkg/cluster"
	"github.com/funkygao/gorequest"
	log "github.com/funkygao/log4go"
)

func (e *Engine) onControllerRebalance(epoch int, decision cluster.Decision) {
	log.Info("[%s] decision: %+v", e.participant, decision)

	for participant, resources := range decision {
		log.Debug("[%s] rpc-> %s %+v", e.participant, participant.Endpoint, resources)

		// edge case:
		// participant might die
		// participant not die, but its Input plugin panic
		// leader might die
		// rpc might be rejected
		statusCode := e.callRPC(participant.Endpoint, epoch, resources)
		switch statusCode {
		case http.StatusOK:
			log.Trace("[%s] rpc<- ok %s", e.participant, participant.Endpoint)

		case http.StatusGone:
			// e,g.
			// resource changed, live participant [1, 2, 3], when RPC sending, p[1] gone
			// just wait for another rebalance event
			log.Warn("[%s] rpc<- %s gone", e.participant, participant.Endpoint)
			return

		case http.StatusBadRequest:
			// should never happen
			log.Critical("[%s] rpc<- %s bad request", e.participant, participant.Endpoint)

		case http.StatusNotAcceptable:
			log.Error("[%s] rpc<- %s leader moved", e.participant, participant.Endpoint)
			return

		default:
			// TODO
			log.Error("[%s] rpc<- %s %d", e.participant, participant.Endpoint, statusCode)
		}

	}
}

func (e *Engine) callRPC(endpoint string, epoch int, resources []cluster.Resource) int {
	resp, _, errs := gorequest.New().
		Post(fmt.Sprintf("http://%s/v1/rebalance?epoch=%d", endpoint, epoch)).
		Set("User-Agent", fmt.Sprintf("dbus-%s", dbus.Revision)).
		SendString(string(cluster.Resources(resources).Marshal())).
		End()
	if len(errs) > 0 {
		// e,g. participant gone
		// connection reset
		// connnection refused
		// FIXME what if connection timeout?
		return http.StatusGone
	}

	return resp.StatusCode
}
