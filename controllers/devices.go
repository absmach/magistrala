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

	"github.com/mainflux/mainflux/db"
	"github.com/mainflux/mainflux/models"

	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"

	"io"
	"net/http"
	"io/ioutil"

	"github.com/go-zoo/bone"
)

/** == Functions == */
/**
 * CreateDevice ()
 */
func CreateDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data, err := ioutil.ReadAll(r.Body)
    if err != nil {
		println("HERE")
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

	/*
	if validateJsonSchema("device", body) != true {
		println("Invalid schema")
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "invalid json schema in request"}`
		io.WriteString(w, str)
		return
	}
	*/

	// Init new Mongo session
	// and get the "devices" collection
	// from this new session
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	// Set up defaults and pick up new values from user-provided JSON
	d := models.Device{Name: "Some Name", Online: false}
	if err := json.Unmarshal(data, &d); err != nil {
		println("Cannot decode!")
		log.Print(err.Error())
		panic(err)
	}

	// Creating UUID Version 4
	uuid := uuid.NewV4()
	fmt.Println(uuid.String())

	d.Id = uuid.String()

	// Timestamp
	t := time.Now().UTC().Format(time.RFC3339)
	d.Created, d.Updated = t, t

	// Insert Device
	if err := Db.C("devices").Insert(d); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		str := `{"response": "cannot create device"}`
		io.WriteString(w, str)
		return
	}

	w.WriteHeader(http.StatusOK)
	str := `{"response": "created", "id": "` + d.Id + `"}`
    io.WriteString(w, str)
}

/**
 * GetDevices()
 */
func GetDevices(w http.ResponseWriter, r *http.Request) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	results := []models.Device{}
	if err := Db.C("devices").Find(nil).All(&results); err != nil {
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
 * GetDevice()
 */
func GetDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := bone.GetValue(r, "device_id")

	result := models.Device{}
	err := Db.C("devices").Find(bson.M{"id": id}).One(&result)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		str := `{"response": "not found", "id": "` + id + `"}`
		if err != nil {
			log.Print(err)
		}
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
 * UpdateDevice()
 */
func UpdateDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data, err := ioutil.ReadAll(r.Body)
    if err != nil {
		println("HERE")
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

	/*
	if validateJsonSchema("device", body) != true {
		println("Invalid schema")
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "invalid json schema in request"}`
		io.WriteString(w, str)
		return
	}
	*/


	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := bone.GetValue(r, "device_id")

	// Check if someone is trying to change "id" key
	// and protect us from this
	if _, ok := body["id"]; ok {
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "invalid request: device id is read-only"}`
		io.WriteString(w, str)
		return
	}
	if _, ok := body["created"]; ok {
		println("Error: can not change device")
		w.WriteHeader(http.StatusBadRequest)
		str := `{"response": "invalid request: 'created' is read-only"}`
		io.WriteString(w, str)
		return
	}

	// Timestamp
	t := time.Now().UTC().Format(time.RFC3339)
	body["updated"] = t

	colQuerier := bson.M{"id": id}
	change := bson.M{"$set": body}
	if err := Db.C("devices").Update(colQuerier, change); err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		str := `{"response": "not updated", "id": "` + id + `"}`
		io.WriteString(w, str)
		return
	}

	w.WriteHeader(http.StatusOK)
	str := `{"response": "updated", "id": "` + id + `"}`
	io.WriteString(w, str)
}

/**
 * DeleteDevice()
 */
func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := bone.GetValue(r, "device_id")

	err := Db.C("devices").Remove(bson.M{"id": id})
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
