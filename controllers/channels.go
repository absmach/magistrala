/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"strconv"

	"github.com/mainflux/mainflux/db"
	"github.com/mainflux/mainflux/models"
	"github.com/mainflux/mainflux/clients"

	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"

	"io"
	"net/http"
	"io/ioutil"

	"github.com/go-zoo/bone"
)

/** == Functions == */

/**
 * CreateChannel ()
 */
func CreateChannel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data, err := ioutil.ReadAll(r.Body)
    if err != nil {
        panic(err)
    }

	if len(data) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "no data (Device ID) provided"}`
		io.WriteString(w, str)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(data, &body); err != nil {
		panic(err)
	}

	/**
	if validateJsonSchema("channel", body) != true {
		println("Invalid schema")
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "invalid json schema in request"}`
		io.WriteString(w, str)
		return
	}
	**/

	// Init new Mongo session
	// and get the "channels" collection
	// from this new session
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	c := models.Channel{}
	if err := json.Unmarshal(data, &c); err != nil {
		panic(err)
	}

	// Creating UUID Version 4
	uuid := uuid.NewV4()
	fmt.Println(uuid.String())

	c.Id = uuid.String()

	// Insert reference to DeviceId
	if len(c.Device) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "no device ID provided in request"}`
		io.WriteString(w, str)
		return
	}

	// TODO Check if Device ID is valid (in database)

	// Timestamp
	t := time.Now().UTC().Format(time.RFC3339)
	c.Created, c.Updated = t, t

	// Insert Channel
	if err := Db.C("channels").Insert(c); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		str := `{"response": "cannot create channel"}`
		io.WriteString(w, str)
		return
	}

	w.WriteHeader(http.StatusOK)
	str := `{"response": "created", "id": "` + c.Id + `"}`
    io.WriteString(w, str)
}

/**
 * GetChannels()
 */
func GetChannels(w http.ResponseWriter, r *http.Request) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	// Get fileter values from parameters:
	// - climit = count limit, limits number of returned `channel` elements
	// - vlimit = value limit, limits number of values within the channel
	var climit, vlimit int
	var err error
	s := r.URL.Query().Get("climit")
	if len(s) == 0 {
		// Set default limit to -5
		climit = -100
	} else {
		climit, err = strconv.Atoi(s); if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			str := `{"response": "wrong count limit"}`
			io.WriteString(w, str)
			return
		}
	}

	s = r.URL.Query().Get("vlimit")
	if len(s) == 0 {
		// Set default limit to -5
		vlimit = -100
	} else {
		vlimit, err = strconv.Atoi(s); if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			str := `{"response": "wrong value limit"}`
			io.WriteString(w, str)
			return
		}
	}

	// Query DB
	results := []models.Channel{}
	if err := Db.C("channels").Find(nil).
				Select(bson.M{"values": bson.M{"$slice": vlimit}}).
				Sort("-_id").Limit(climit).All(&results); err != nil {
		log.Print(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	res, err := json.Marshal(results)
	if err != nil {
		log.Print(err)
	}
    io.WriteString(w, string(res))
}

/**
 * GetChannel()
 */
func GetChannel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := bone.GetValue(r, "channel_id")


	var vlimit int
	var err error
	s := r.URL.Query().Get("vlimit")
	if len(s) == 0 {
		// Set default limit to -5
		vlimit = -5
	} else {
		vlimit, err = strconv.Atoi(s); if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			str := `{"response": "wrong limit"}`
			io.WriteString(w, str)
			return
		}
	}

	result := models.Channel{}
	if err := Db.C("channels").Find(bson.M{"id": id}).
				Select(bson.M{"values": bson.M{"$slice": vlimit}}).
				One(&result); err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		str := `{"response": "not found", "id": "` + id + `"}`
		io.WriteString(w, str)
		return
	}


	w.WriteHeader(http.StatusOK)
	res, err := json.Marshal(result)
	if err != nil {
		log.Print(err)
	}
    io.WriteString(w, string(res))
}

/**
 * UpdateChannel()
 */
func UpdateChannel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	data, err := ioutil.ReadAll(r.Body)
    if err != nil {
        panic(err)
    }

	if len(data) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "no data provided"}`
		io.WriteString(w, str)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(data, &body); err != nil {
		panic(err)
	}

	/**
	if validateJsonSchema("channel", body) != true {
		println("Invalid schema")
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "invalid json schema in request"}`
		io.WriteString(w, str)
		return
	}
	**/

	id := bone.GetValue(r, "channel_id")

	// Publish the channel update.
	// This will be catched by the MQTT main client (subscribed to all channel topics)
	// and then written in the DB in the MQTT handler
	token := clients.MqttClient.Publish("mainflux/" + id, 0, false, string(data))
	token.Wait()

	// Wait on status from MQTT handler (which executes DB write)
	status := <-clients.WriteStatusChannel
	w.WriteHeader(status.Nb)
	str := `{"response": "` + status.Str + `"}`
	io.WriteString(w, str)
}

/**
 * DeleteChannel()
 */
func DeleteChannel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := bone.GetValue(r, "channel_id")

	err := Db.C("channels").Remove(bson.M{"id": id})
		if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		str := `{"response": "not deleted", "id": "` + id + `"}`
		io.WriteString(w, str)
		return
	}

	w.WriteHeader(http.StatusOK)
	str := `{"response": "deleted", "id": "` + id + `"}`
    io.WriteString(w, str)
}


