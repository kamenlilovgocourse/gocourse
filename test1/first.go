package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var Global int

func init() {
	Global = 16
}

func ProbaSum(a int, b int) int {
	sum := a + b
	return sum
}

type ResponseJSON struct {
	Current CurrentJSON `json:"current"`
}

type CurrentJSON struct {
	TempC     float64       `json:"temp_c"`
	Condition ConditionJSON `json:"condition"`
}

type ConditionJSON struct {
	Text string `json:"text"`
}

type Weather struct {
	TempC   float64
	Details string
}

type Sinoptik struct {
	// w map[string]Weather
	api string
}

func NewSinoptik() *Sinoptik {
	return &Sinoptik{
		api: "https://api.weatherapi.com/v1/current.json?key=964949da99d949f5b42113942221912",
	}
}

func (s *Sinoptik) Weather(city string) (Weather, error) {
	resp, err := http.Get(s.api + "&q=" + city)
	if err != nil {
		return Weather{}, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Weather{}, err
	}

	r := ResponseJSON{}
	_ = json.Unmarshal(bodyBytes, &r)

	return Weather{
		TempC:   r.Current.TempC,
		Details: r.Current.Condition.Text,
	}, nil
}

type Assistant struct {
}

func (a *Assistant) NeedUmbrella() bool {
	return false
}

func main() {
	city := "Sofia"
	sinoptik := NewSinoptik()
	w, err := sinoptik.Weather(city)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	fmt.Printf("Weather in %s is %f degrees : %s", city, w.TempC, w.Details)
}
