package sqlite_db

import (
	"fmt"
	"log"
	"testing"
)

func TestInitDB(t *testing.T) {
	err := InitDB("/Users/admin/go/empty/weather.db")
	if err != nil {
		log.Fatalf("error querying users: %v", err)
		return
	}
	defer CloseDB()
	type cityInfo struct {
		Id          int     `json:"id" db:"id"`
		CityName    string  `json:"city_name" db:"city_name"`
		StateName   string  `json:"state_name" db:"state_name"`
		CountryName string  `json:"country_name" db:"country_name"`
		Latitude    float64 `json:"latitude" db:"latitude"`
		Longitude   float64 `json:"longitude" db:"longitude"`
		Iso2        string  `json:"iso2" db:"iso2"`
	}
	data := []cityInfo{}
	err = Db().Select(&data, "select a.id,a.name as city_name ,b.name as state_name,c.name as country_name,a.latitude,a.longitude,c.iso2 from cities_id a inner join states b on a.state_id = b.id inner join countries c on c.id = b.country_id where a.name like 'Raleigh%'")
	if err != nil {
		log.Fatalf("error querying users: %v", err)
	}

	for i, i2 := range data {
		fmt.Printf("%d,%+v\n", i, i2)
	}
}

type CityInfo struct {
	Id       int    `json:"id" db:"id"`
	CityName string `json:"city_name" db:"city_name"`
}

type CityInfoAll struct {
	Table      string
	targetLang string
	CityInfo
}
