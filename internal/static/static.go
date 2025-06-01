package static

import (
	"CandallGo/internal/db"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// State
// Action's: - "delete" - delete this msg, Data - msg.id
type State struct {
	Action string `json:"action"`
	Data   string `json:"data"` // Дата зависит от Action и должна подходить под него
}

type PaymentState struct {
	Action string         `json:"action"`
	Data   db.PaymentData `json:"data"` // Дата зависит от Action и должна подходить под него
}

func EncodeState(state State) (string, error) {
	dataJson, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	stateString := string(dataJson)
	return stateString, nil
}

func EncodePayment(state PaymentState) (string, error) {
	var newData = fmt.Sprintf("%s_%d_%d", state.Data.GroupId, state.Data.CurrencyId, state.Data.ProductId)
	newState := State{
		Action: state.Action, Data: newData,
	}
	dataJson, err := json.Marshal(newState)
	if err != nil {
		return "", err
	}
	stateString := string(dataJson)
	return stateString, nil
}

func DecodeState(stateString string) (State, error) {
	var state State
	err := json.Unmarshal([]byte(stateString), &state)
	return state, err
}

func DecodePayment(stateString string) (PaymentState, error) {
	var newState PaymentState
	var state State
	err := json.Unmarshal([]byte(stateString), &state)

	newState.Action = state.Action
	newState.Data.GroupId = strings.Split(state.Data, "_")[0]
	newState.Data.CurrencyId, err = strconv.Atoi(strings.Split(state.Data, "_")[1])
	if err != nil {
		return PaymentState{}, err
	}
	newState.Data.ProductId, err = strconv.Atoi(strings.Split(state.Data, "_")[2])
	if err != nil {
		return PaymentState{}, err
	}

	return newState, err
}
