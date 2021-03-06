package endpoints

import (
	"encoding/json"
	"time"

	"github.com/Proofsuite/amp-matching-engine/interfaces"
	"github.com/Proofsuite/amp-matching-engine/types"
	"github.com/Proofsuite/amp-matching-engine/ws"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-ozzo/ozzo-routing"
)

type OHLCVEndpoint struct {
	ohlcvService interfaces.OHLCVService
}

func ServeOHLCVResource(
	rg *routing.RouteGroup,
	ohlcvService interfaces.OHLCVService,
) {
	e := &OHLCVEndpoint{ohlcvService}
	rg.Post("/ohlcv", e.ohlcv)
	ws.RegisterChannel(ws.OHLCVChannel, e.ohlcvWebSocket)
}

func (e *OHLCVEndpoint) ohlcv(c *routing.Context) error {
	var model types.TickRequest
	if err := c.Read(&model); err != nil {
		return err
	}

	if model.Units == "" {
		model.Units = "hour"
	}

	if model.Duration == 0 {
		model.Duration = 24
	}

	if model.To == 0 {
		model.To = time.Now().Unix()
	}

	res, err := e.ohlcvService.GetOHLCV(model.Pair, model.Duration, model.Units, model.From, model.To)
	if err != nil {
		return err
	}

	return c.Write(res)
}

func (e *OHLCVEndpoint) ohlcvWebSocket(input interface{}, conn *ws.Conn) {
	startTs := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

	mab, _ := json.Marshal(input)
	var payload *types.WebSocketPayload

	err := json.Unmarshal(mab, &payload)
	if err != nil {
		logger.Error(err)
	}

	socket := ws.GetOHLCVSocket()

	if payload.Type != "subscription" {
		logger.Error("Payload is not subscription type")
		socket.SendErrorMessage(conn, "Payload is not of subscription type")
		return
	}

	dab, _ := json.Marshal(payload.Data)
	var msg *types.WebSocketSubscription

	err = json.Unmarshal(dab, &msg)
	if err != nil {
		logger.Error(err)
	}

	if (msg.Pair.BaseToken == common.Address{}) {
		message := map[string]string{
			"Code":    "Invalid_Pair_BaseToken",
			"Message": "Invalid Pair BaseToken passed in Params",
		}

		socket.SendErrorMessage(conn, message)
		return
	}

	if (msg.Pair.QuoteToken == common.Address{}) {
		message := map[string]string{
			"Code":    "Invalid_Pair_BaseToken",
			"Message": "Invalid Pair BaseToken passed in Params",
		}

		socket.SendErrorMessage(conn, message)
		return
	}

	if msg.Params.From == 0 {
		msg.Params.From = startTs.Unix()
	}

	if msg.Params.To == 0 {
		msg.Params.To = time.Now().Unix()
	}

	if msg.Params.Duration == 0 {
		msg.Params.Duration = 24
	}

	if msg.Params.Units == "" {
		msg.Params.Units = "hour"
	}

	if msg.Event == types.SUBSCRIBE {
		e.ohlcvService.Subscribe(conn, msg.Pair.BaseToken, msg.Pair.QuoteToken, &msg.Params)
	}

	if msg.Event == types.UNSUBSCRIBE {
		e.ohlcvService.Unsubscribe(conn, msg.Pair.BaseToken, msg.Pair.QuoteToken, &msg.Params)
	}
}
