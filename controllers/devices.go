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

	"github.com/mainflux/mainflux-lite/db"
	"github.com/mainflux/mainflux-lite/models"

	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"

	"github.com/kataras/iris"
)

/** == Functions == */
/**
 * CreateDevice ()
 */
func CreateDevice(ctx *iris.Context) {
	var body map[string]interface{}
	ctx.ReadJSON(&body)
	if validateJsonSchema("device", body) != true {
		println("Invalid schema")
		ctx.JSON(iris.StatusBadRequest, iris.Map{"response": "invalid json schema in request"})
		return
	}

	// Init new Mongo session
	// and get the "devices" collection
	// from this new session
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	// Set up defaults and pick up new values from user-provided JSON
	d := models.Device{Name: "Some Name"}
	json.Unmarshal(ctx.RequestCtx.Request.Body(), &d)

	// Creating UUID Version 4
	uuid := uuid.NewV4()
	fmt.Println(uuid.String())

	d.Id = uuid.String()

	// Timestamp
	t := time.Now().UTC().Format(time.RFC3339)
	d.Created, d.Updated = t, t

	// Insert Device
	erri := Db.C("devices").Insert(d)
	if erri != nil {
		ctx.JSON(iris.StatusInternalServerError, iris.Map{"response": "cannot create device"})
		return
	}

	ctx.JSON(iris.StatusCreated, iris.Map{"response": "created", "id": d.Id})
}

/**
 * GetDevices()
 */
func GetDevices(ctx *iris.Context) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	results := []models.Device{}
	err := Db.C("devices").Find(nil).All(&results)
	if err != nil {
		log.Print(err)
	}

	ctx.JSON(iris.StatusOK, &results)
}

/**
 * GetDevice()
 */
func GetDevice(ctx *iris.Context) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := ctx.Param("device_id")

	result := models.Device{}
	err := Db.C("devices").Find(bson.M{"id": id}).One(&result)
	if err != nil {
		log.Print(err)
		ctx.JSON(iris.StatusNotFound, iris.Map{"response": "not found", "id": id})
		return
	}

	ctx.JSON(iris.StatusOK, &result)
}

/**
 * UpdateDevice()
 */
func UpdateDevice(ctx *iris.Context) {
	var body map[string]interface{}
	ctx.ReadJSON(&body)

	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := ctx.Param("device_id")

	// Validate JSON schema user provided
	if validateJsonSchema("device", body) != true {
		println("Invalid schema")
		ctx.JSON(iris.StatusBadRequest, iris.Map{"response": "invalid json schema in request"})
		return
	}

	// Check if someone is trying to change "id" key
	// and protect us from this
	if _, ok := body["id"]; ok {
		ctx.JSON(iris.StatusBadRequest, iris.Map{"response": "invalid request: device id is read-only"})
		return
	}
	if _, ok := body["created"]; ok {
		println("Error: can not change device")
		ctx.JSON(iris.StatusBadRequest, iris.Map{"response": "invalid request: 'created' is read-only"})
		return
	}

	// Timestamp
	t := time.Now().UTC().Format(time.RFC3339)
	body["updated"] = t

	colQuerier := bson.M{"id": id}
	change := bson.M{"$set": body}
	err := Db.C("devices").Update(colQuerier, change)
	if err != nil {
		log.Print(err)
		ctx.JSON(iris.StatusNotFound, iris.Map{"response": "not updated", "id": id})
		return
	}

	ctx.JSON(iris.StatusOK, iris.Map{"response": "updated", "id": id})
}

/**
 * DeleteDevice()
 */
func DeleteDevice(ctx *iris.Context) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := ctx.Param("device_id")

	err := Db.C("devices").Remove(bson.M{"id": id})
	if err != nil {
		log.Print(err)
		ctx.JSON(iris.StatusNotFound, iris.Map{"response": "not deleted", "id": id})
		return
	}

	ctx.JSON(iris.StatusOK, iris.Map{"response": "deleted", "id": id})
}
