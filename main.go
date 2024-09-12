package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	SeqNum = 0
)

var (
	hturl = url.URL{
		Scheme: "https",
		Host:   "firestore.googleapis.com",
		Path:   "/v1/projects/junctor-hackertracker/databases/(default)/documents/conferences/MOCA2024:runQuery",
	}
)

type StringField struct {
	StringValue string `json:"stringValue,omitempty"`
}

type IntField struct {
	IntegerValue string `json:"integerValue,omitempty"`
}

type Value struct {
	MapValue struct {
		Fields struct {
			ConferenceID interface{} `json:"conference_id,omitempty"`
			EventIDs     ArrayField  `json:"event_i_ds,omitempty"`
			Name         StringField `json:"name,omitempty"`
			Links        ArrayField  `json:"links,omitempty"`
			Affiliations ArrayField  `json:"affiliations,omitempty"`
			Media        ArrayField  `json:"media,omitempty"`
			ID           IntField    `json:"id,omitempty"`
			ShortName    StringField `json:"short_name,omitempty"`
		} `json:"fields,omitempty"`
	} `json:"mapValue,omitempty"`
}

type ArrayField struct {
	ArrayValue struct {
		Values []Value `json:"values,omitempty"`
	} `json:"arrayValue,omitempty"`
}

type MapField struct {
	MapValue struct {
		Fields struct {
			ConferenceID IntField    `json:"conference_id,omitempty"`
			Conference   StringField `json:"conference,omitempty"`
			UpdatedAt    StringField `json:"updated_at,omitempty"`
			UpdatedTsz   StringField `json:"updated_tsz,omitempty"`
			Color        StringField `json:"color,omitempty"`
			Name         StringField `json:"name,omitempty"`
			ID           IntField    `json:"id,omitempty"`
		} `json:"fields,omitempty"`
	} `json:"mapValue,omitempty"`
}

type Item struct {
	Document struct {
		Name   string `json:"name"`
		Fields struct {
			Conference       StringField `json:"conference,omitempty"`
			Timezone         StringField `json:"timezone,omitempty"`
			Link             StringField `json:"link,omitempty"`
			Title            StringField `json:"title,omitempty"`
			Description      StringField `json:"description,omitempty"`
			Media            ArrayField  `json:"media,omitempty"`
			Type             MapField    `json:"type,omitempty"`
			BeginTsz         StringField `json:"begin_tsz,omitempty"`
			EndTsz           StringField `json:"end_tsz,omitempty"`
			BeginTimestamp   StringField `json:"begin_timestamp,omitempty"`
			EndTimestamp     StringField `json:"end_timestamp,omitempty"`
			UpdatedTsz       StringField `json:"updated_tsz,omitempty"`
			UpdatedTimestamp StringField `json:"updated_timestamp,omitempty"`
			Location         Value       `json:"location,omitempty"`
			Speakers         ArrayField  `json:"speakers,omitempty"`
			Links            ArrayField  `json:"links,omitempty"`
		} `json:"fields,omitempty"`
	} `json:"document"`
}

func main() {
	reqData := []byte(`{
  "structuredQuery": {
    "from": [
      {
        "collectionId": "events"
      }
    ],
    "orderBy": [
      {
        "field": {
          "fieldPath": "begin_timestamp"
        },
        "direction": "DESCENDING"
      },
      {
        "field": {
          "fieldPath": "__name__"
        },
        "direction": "DESCENDING"
      }
    ]
  }
}`)
	req, err := http.NewRequest(http.MethodPost, hturl.String(), bytes.NewBuffer(reqData))
	if err != nil {
		logrus.Fatalf("Failed to create new request: %v", err)
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Fatalf("HTTP POST failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		logrus.Fatalf("Status code is not 200 OK but %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatalf("Failed to read HTTP body: %v", err)
	}
	var items []Item
	if err := json.Unmarshal(body, &items); err != nil {
		logrus.Fatalf("Failed to unmarshal response: %v", err)
	}
	ics := fmt.Sprintf(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//moca/camp/schedule v1.0//IT
VTIMEZONE: Europe/Rome
`)

	for _, item := range items {
		eventUID := item.Document.Fields.Type.MapValue.Fields.ID.IntegerValue
		location := item.Document.Fields.Location.MapValue.Fields.Name.StringValue
		speakerNames := make([]string, 0, len(item.Document.Fields.Speakers.ArrayValue.Values))
		for _, v := range item.Document.Fields.Speakers.ArrayValue.Values {
			name := v.MapValue.Fields.Name.StringValue
			if name != "" {
				name = strings.Replace(name, "\n", "\\n", -1)
				name = strings.Replace(name, "\"", "'", -1)
				speakerNames = append(speakerNames, name)
			}
		}
		speakers := "unknown"
		if len(speakerNames) > 0 {
			speakers = strings.Join(speakerNames, ", ")
		}
		start, err := time.Parse("2006-01-02T15:04:05Z", item.Document.Fields.BeginTsz.StringValue)
		if err != nil {
			logrus.Fatalf("Failed to parse start time %q: %v", item.Document.Fields.BeginTsz.StringValue, err)
		}
		end, err := time.Parse("2006-01-02T15:04:05Z", item.Document.Fields.EndTsz.StringValue)
		if err != nil {
			logrus.Fatalf("Failed to parse end time %q: %v", item.Document.Fields.EndTsz.StringValue, err)
		}
		summary := item.Document.Fields.Title.StringValue
		description := item.Document.Fields.Description.StringValue
		description = strings.Replace(description, "\n", "\\n", -1)
		description = strings.Replace(description, "\r", "", -1)
		ics += fmt.Sprintf(`BEGIN:VEVENT
UID:%s
SEQUENCE: %d
ORGANIZER;CN=%s:MAILTO:info@olografix.org
DTSTAMP:20240912T140000Z
DTSTART:%s
DTEND:%s
SUMMARY:%s
DESCRIPTION: %s
GEO:42.5816338;14.0901461
LOCATION: %s
END:VEVENT
`,
			eventUID,
			SeqNum,
			speakers,
			start.Format("20060102T150405Z"),
			end.Format("20060102T150405Z"),
			summary,
			description,
			location,
		)
	}
	ics += fmt.Sprintln("END:VCALENDAR")
	ics = strings.Replace(ics, "\n", "\r\n", -1)
	fmt.Println(ics)
}
