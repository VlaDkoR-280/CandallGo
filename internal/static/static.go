package static

import "encoding/json"

type State struct {
	Action string `json:"action"`
	Data   string `json:"data"` // Дата зависит от Action и должна подходить под него
}

func EncodeState(state State) (string, error) {
	dataJson, err := json.Marshal(state)
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
