// Project Gonder.
// Author Supme
// Copyright Supme 2016
// License http://opensource.org/licenses/MIT MIT License	
//
//  THE SOFTWARE AND DOCUMENTATION ARE PROVIDED "AS IS" WITHOUT WARRANTY OF
//  ANY KIND, EITHER EXPRESSED OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
//  IMPLIED WARRANTIES OF MERCHANTABILITY AND/OR FITNESS FOR A PARTICULAR
//  PURPOSE.
//
// Please see the License.txt file for more information.
//
package api

import (
	"net/http"
	"encoding/json"
	"github.com/supme/gonder/models"
	"encoding/base64"
	"io/ioutil"
	"time"
	"os"
	"encoding/csv"
)

type Recipient struct {
	Id   int64 `json:"recid"`
	Name string `json:"name"`
	Email string `json:"email"`
	Result string `json:"result"`
}

type Recipients struct {
	Total	    int `json:"total"`
	Records		[]Recipient `json:"records"`
}

type RecipientParam struct  {
	Key string `json:"key"`
	Value string `json:"value"`
}

type RecipientParams struct {
	Total	    int `json:"total"`
	Records		[]RecipientParam `json:"records"`
}


func recipients(w http.ResponseWriter, r *http.Request)  {
	var err error
	var js []byte

	if err = r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Form["content"][0] == "recipients" {
		switch r.Form["cmd"][0] {
		case "get-records":
			if auth.Right("get-recipients") && auth.CampaignRightString(r.Form["campaign"][0]) {
				rs, err := getRecipients( r.Form["campaign"][0], r.Form["offset"][0], r.Form["limit"][0])
				js, err = json.Marshal(rs)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}  else {
				js = []byte(`{"status": "error", "message": "Forbidden get recipients"}`)
			}
			break

		case "upload":
			if auth.Right("upload-recipients") && auth.CampaignRightString(r.Form["campaign"][0]) {
				content, err := base64.StdEncoding.DecodeString(r.FormValue("base64"))
				if err != nil {
					js = []byte(`{"status": "Error", "message": "Base64 decode"}`)
				}
				file := "./tmp/" + time.Now().String()
				err = ioutil.WriteFile(file, content, 0644)
				if err != nil {
					js = []byte(`{"status": "Error", "message": "Write file"}`)
				}

				err = recipientCsv(r.FormValue("campaign"), file)
				if err != nil {
					js = []byte(`{"status": "Error", "message": "Add recipients csv"}`)
				}
			} else {
				js = []byte(`{"status": "error", "message": "Forbidden upload recipients"}`)
			}
			break

		case "deleteAll":
			if auth.Right("delete-recipients") && auth.CampaignRightString(r.Form["campaign"][0]) {
				err = delRecipients(r.Form["campaign"][0])
				if err != nil {
					js = []byte(`{"status": "Error", "message": "Can't delete all recipients"}`)
				}
			} else {
				js = []byte(`{"status": "error", "message": "Forbidden delete recipients"}`)
			}
		}
	}

	if r.Form["content"][0] == "parameters" {
		switch r.Form["cmd"][0] {
		case "get-records":
			rId, err := getRecipientCampaign(r.Form["recipient"][0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if auth.Right("get-recipient-parameters") && auth.CampaignRight(rId) {
				ps, err := getRecipientParams( r.Form["recipient"][0], r.Form["offset"][0], r.Form["limit"][0])
				js, err = json.Marshal(ps)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}  else {
				js = []byte(`{"status": "error", "message": "Forbidden get recipient parameters"}`)
			}

			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func getRecipientCampaign(recipientId string) (int64, error){
	var id int64
	err := models.Db.QueryRow("SELECT `campaign_id` FROM `recipient` WHERE `id`=?", recipientId).Scan(&id)
	return id, err
}

//ToDo check right errors
func getRecipients(campaign, offset, limit string) (Recipients, error) {
	var err error
	var r Recipient
	var rs Recipients
	rs.Records = []Recipient{}
	query, err := models.Db.Query("SELECT `id`, `name`, `email`, `status` FROM `recipient` WHERE `removed`!=1 AND `campaign_id`=?  LIMIT ? OFFSET ?", campaign, limit, offset)
	if err != nil {
		return rs, err
	}
	defer query.Close()
	for query.Next() {
		err = query.Scan(&r.Id, &r.Name, &r.Email, &r.Result)

		rs.Records = append(rs.Records, r)
	}
	err = models.Db.QueryRow("SELECT COUNT(*) FROM `recipient` WHERE `removed`!=1 AND `campaign_id`=?", campaign).Scan(&rs.Total)
	return rs, nil

}

//ToDo check right errors
func getRecipientParams(recipient, offset, limit string) (RecipientParams, error) {
	var err error
	var p RecipientParam
	var ps RecipientParams
	ps.Records = []RecipientParam{}
	query, err := models.Db.Query("SELECT `key`, `value` FROM `parameter` WHERE `recipient_id`=?", recipient)
	if err != nil {
		return ps, err
	}
	defer query.Close()
	for query.Next() {
		err = query.Scan(&p.Key, &p.Value)
		ps.Records = append(ps.Records, p)
	}
	err = models.Db.QueryRow("SELECT COUNT(*) FROM `parameter` WHERE `recipient_id`=?", recipient).Scan(&ps.Total)
	return ps, err
}

func delRecipients(campaignId string) error {
	_, err := models.Db.Exec("UPDATE `recipient` SET `removed`=1 WHERE `campaign_id`=?", campaignId)
	return err
}


// ToDo optimize this
func recipientCsv(campaignId string, file string) error {
//	var groupId string
//	models.Db.QueryRow("SELECT `group_id` FROM `campaign` WHERE `id`=? ", campaignId).Scan(&groupId)

	title := make(map[int]string)
	data := make(map[string]string)

	csvfile, err := os.Open(file)
	if err != nil {
		return err
	}

	reader := csv.NewReader(csvfile)
	reader.FieldsPerRecord = -1
	rawCSVdata, err := reader.ReadAll()
	if err != nil {
		return err
	}
	for k, v := range rawCSVdata {
		if k == 0 {
			for i, t := range v {
				title[i] = t
			}
		} else {
			email := ""
			name := ""
			for i, t := range v {
				if i == 0 {
					email = t
				} else if i == 1 {
					name = t
				} else {
					data[title[i]] = t
				}
			}

//			var cnt int
//			models.Db.QueryRow("SELECT COUNT(*) FROM `unsubscribe` WHERE `group_id`=? AND `email`=?", groupId, email).Scan(&cnt)

			sql := "INSERT INTO recipient (`campaign_id`, `email`, `name`) VALUES (?, ?, ?)"
//			if cnt > 0 {
//				sql = "INSERT INTO recipient (`campaign_id`, `email`, `name`, `status`) VALUES (?, ?, ?, 'Unsubscribed')"
//			}

			res, err := models.Db.Exec(sql, campaignId, email, name)
			if err != nil {
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}
			for i, t := range data {
				_, err := models.Db.Exec("INSERT INTO parameter (`recipient_id`, `key`, `value`) VALUES (?, ?, ?)", id, i, t)
				if err != nil {
					return err
				}
			}
		}
	}

	csvfile.Close()

	os.Remove(file)

	return err
}