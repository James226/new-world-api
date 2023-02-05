package main

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

type State struct {
	Position Point
}

type StateManager struct {
	secret []byte
}

func NewStateManager(secret []byte) *StateManager {
	return &StateManager{
		secret: secret,
	}
}

func (r *StateManager) Deserialize(state string) (*State, error) {
	token, err := jwt.Parse(state, func(jwtToken *jwt.Token) (interface{}, error) {
		if _, ok := jwtToken.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected method: %s", jwtToken.Header["alg"])
		}

		return r.secret, nil
	})

	if !token.Valid {
		return nil, errors.New("invalid state provided")
	}

	if err != nil {
		return nil, err
	}

	claims := token.Claims.(jwt.MapClaims)
	position := claims["position"].(map[string]int)
	s := State{
		Position: Point{
			X: position["x"],
			Y: position["y"],
			Z: position["z"],
		},
	}

	return &s, nil
}

func (r *StateManager) Serialize(state State) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"position": state.Position,
		"nbf":      time.Now().Unix(),
	})

	return token.SignedString(r.secret)
}
