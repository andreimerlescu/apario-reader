package main

import (
	`context`
	`log`
	`strconv`
	`strings`
)

func processLocation(ctx context.Context, row []Column) error {
	var (
		countryName, countryCode, continent, stateProvinceName, cityName string
		latitude, longitude                                              float64
		intConversionErr, floatConversionErr                             error
	)
	for _, column := range row {
		switch column.Header {
		case "countryname":
			countryName = column.Value
		case "countrycode":
			countryCode = column.Value
		case "Continent":
			continent = column.Value
		case "StateProvinceName":
			stateProvinceName = column.Value
		case "cityname":
			cityName = column.Value
		case "latitude":
			if len(column.Value) > 0 {
				latitude, floatConversionErr = strconv.ParseFloat(column.Value, 32)
			}
		case "longitude":
			if len(column.Value) > 0 {
				longitude, floatConversionErr = strconv.ParseFloat(column.Value, 32)
			}
		}
	}

	if intConversionErr != nil {
		log.Fatalf("failed to convert int: %v", intConversionErr)
	}

	if floatConversionErr != nil {
		log.Fatalf("failed to convert float: %v", floatConversionErr)
	}

	location := &Location{
		Continent:   continent,
		Country:     countryName,
		CountryCode: countryCode,
		City:        cityName,
		State:       stateProvinceName,
		Longitude:   longitude,
		Latitude:    latitude,
	}

	cityName = strings.ToLower(cityName)
	mu_location_cities.Lock()
	m_location_cities = append(m_location_cities, location)
	mu_location_cities.Unlock()

	countryName = strings.ToLower(countryName)
	mu_location_countries.Lock()
	m_location_countries = append(m_location_countries, location)
	mu_location_countries.Unlock()

	stateProvinceName = strings.ToLower(stateProvinceName)
	mu_location_states.Lock()
	m_location_states = append(m_location_states, location)
	mu_location_states.Unlock()
	return nil
}
