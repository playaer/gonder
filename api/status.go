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
	"github.com/gorilla/websocket"
	"net/http"
	"github.com/hpcloud/tail"
	"os"
)


var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func campaignLog(w http.ResponseWriter, r *http.Request) {
	if auth.Right("get-log-campaign") {
		logHandler(w, r, "./log/campaign.log")
	} else {
		http.Error(w, "Forbidden get log campaign", http.StatusForbidden)
		return
	}
}

func apiLog(w http.ResponseWriter, r *http.Request) {
	if auth.Right("get-log-api") {
		logHandler(w, r, "./log/api.log")
	} else {
		http.Error(w, "Forbidden get log api", http.StatusForbidden)
		return
	}
}

func statisticLog(w http.ResponseWriter, r *http.Request) {
	if auth.Right("get-log-statistic") {
		logHandler(w, r, "./log/statistic.log")
	} else {
		http.Error(w, "Forbidden get log statistic", http.StatusForbidden)
		return
	}
}

func mainLog(w http.ResponseWriter, r *http.Request) {
	if auth.Right("get-log-main") {
		logHandler(w, r, "./log/main.log")
	} else {
		http.Error(w, "Forbidden get log main", http.StatusForbidden)
		return
	}
}

func logHandler(w http.ResponseWriter, r *http.Request, file string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		apilog.Println(err)
		return
	}

	defer conn.Close()

	err = conn.WriteMessage(websocket.TextMessage, []byte("..."));
	if  err != nil {
		apilog.Println(err)
		return
	}

	var offset tail.SeekInfo
	offset.Whence = 2
	fi, err := os.Open(file)
	if err != nil {
		apilog.Println(err)
	}
	f, err := fi.Stat()
	if err != nil {
		apilog.Println(err)
	}
	if f.Size() < 2000 {
		offset.Offset = f.Size() * (-1)
	} else {
		offset.Offset = -2000
	}
	fi.Close()

	conf := tail.Config{
		Follow: true,
		ReOpen: true,
		Location: &offset,
		Logger: tail.DiscardingLogger,
	}
	t, err := tail.TailFile(file, conf)
	if err != nil {
		apilog.Println(err)
	}

	for line := range t.Lines {
		err = conn.WriteMessage(websocket.TextMessage, []byte(line.Text));
		if  err != nil {
			apilog.Println(err)
			return
		}
	}
}