package main

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
)

type target struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

func writeTarget(out *os.File, method, url, bodyJSON string) error {
	t := target{
		Method: method,
		URL:    url,
		Header: map[string][]string{"Content-Type": {"application/json"}},
		Body:   base64.StdEncoding.EncodeToString([]byte(bodyJSON)),
	}
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = out.Write(b)
	return err
}

func main() {
	// checkin
	{
		in, err := os.Open("devices.csv")
		if err != nil {
			panic(err)
		}
		defer in.Close()

		out, err := os.Create("targets-checkin.json")
		if err != nil {
			panic(err)
		}
		defer out.Close()

		r := csv.NewReader(in)
		r.FieldsPerRecord = -1
		n := 0
		for {
			row, err := r.Read()
			if err != nil {
				break
			}
			if len(row) < 3 {
				continue
			}
			// header row
			if row[0] == "device_id" {
				continue
			}
			body := `{"current_version":"` + row[2] + `"}`
			if err := writeTarget(out, "POST",
				"http://localhost:8080/api/v1/devices/"+row[0]+"/checkin",
				body,
			); err != nil {
				panic(err)
			}
			n++
		}
		fmt.Println("checkin targets:", n)
	}

	// report-success
	{
		in, err := os.Open("devices.csv")
		if err != nil {
			panic(err)
		}
		defer in.Close()

		out, err := os.Create("targets-report-success.json")
		if err != nil {
			panic(err)
		}
		defer out.Close()

		r := csv.NewReader(in)
		r.FieldsPerRecord = -1
		n := 0
		for {
			row, err := r.Read()
			if err != nil {
				break
			}
			if len(row) < 5 {
				continue
			}
			if row[0] == "device_id" {
				continue
			}
			body := `{"campaign_id":"` + row[3] + `","stage_id":"` + row[4] + `","result":"success"}`
			if err := writeTarget(out, "POST",
				"http://localhost:8080/api/v1/devices/"+row[0]+"/report",
				body,
			); err != nil {
				panic(err)
			}
			n++
		}
		fmt.Println("report targets:", n)
	}
}
