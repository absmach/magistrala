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
	"github.com/mainflux/mainflux-lite/clients"

	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2/bson"

	"github.com/kataras/iris"
)

/** == Functions == */

/**
 * CreateChannel ()
 */
func CreateChannel(ctx *iris.Context) {
	var body map[string]interface{}
	ctx.ReadJSON(&body)
	/*
	if validateJsonSchema("channel", body) != true {
		println("Invalid schema")
		ctx.JSON(iris.StatusBadRequest, iris.Map{"response": "invalid json schema in request"})
		return
	}
	*/

	// Init new Mongo session
	// and get the "channels" collection
	// from this new session
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()


	c := models.Channel{}
	json.Unmarshal(ctx.RequestCtx.Request.Body(), &c)

	// Creating UUID Version 4
	uuid := uuid.NewV4()
	fmt.Println(uuid.String())

	c.Id = uuid.String()

	// Insert reference to DeviceId
	did := ctx.Param("device_id")
	c.Device = did

	// Timestamp
	t := time.Now().UTC().Format(time.RFC3339)
	c.Created, c.Updated = t, t

	// Insert Channel
	err := Db.C("channels").Insert(c)
	if err != nil {
		ctx.JSON(iris.StatusInternalServerError, iris.Map{"response": "cannot create device"})
		return
	}

	ctx.JSON(iris.StatusCreated, iris.Map{"response": "created", "id": c.Id})
}

/**
 * GetChannels()
 */
func GetChannels(ctx *iris.Context) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	results := []models.Channel{}
	err := Db.C("channels").Find(nil).All(&results)
	if err != nil {
		log.Print(err)
	}

	ctx.JSON(iris.StatusOK, &results)
}

/**
 * GetChannel()
 */
func GetChannel(ctx *iris.Context) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := ctx.Param("channel_id")

	result := models.Channel{}
	err := Db.C("channels").Find(bson.M{"id": id}).One(&result)
	if err != nil {
		log.Print(err)
		ctx.JSON(iris.StatusNotFound, iris.Map{"response": "not found", "id": id})
		return
	}

	ctx.JSON(iris.StatusOK, &result)
}

/**
 * UpdateChannel()
 */
func UpdateChannel(ctx *iris.Context) {
	var body map[string]interface{}
	ctx.ReadJSON(&body)
	// Validate JSON schema user provided
	/*
	if validateJsonSchema("channel", body) != true {
		println("Invalid schema")
		ctx.JSON(iris.StatusBadRequest, iris.Map{"response": "invalid json schema in request"})
		return
	}
	*/

	id := ctx.Param("channel_id")


	// Publish the channel update.
	// This will be catched by the MQTT main client (subscribed to all channel topics)
	// and then written in the DB in the MQTT handler
	token := clients.MqttClient.Publish("mainflux/" + id, 0, false, string(ctx.RequestCtx.Request.Body()))
	token.Wait()

	// Wait on status from MQTT handler (which executes DB write)
	status := <-clients.WriteStatusChannel
	ctx.JSON(status.Nb, iris.Map{"response": status.Str})
}

/**
 * DeleteChannel()
 */
func DeleteChannel(ctx *iris.Context) {
	Db := db.MgoDb{}
	Db.Init()
	defer Db.Close()

	id := ctx.Param("channel_id")

	err := Db.C("channels").Remove(bson.M{"id": id})
		if err != nil {
		log.Print(err)
		ctx.JSON(iris.StatusNotFound, iris.Map{"response": "not deleted", "id": id})
		return
	}

	ctx.JSON(iris.StatusOK, iris.Map{"response": "deleted", "id": id})
}


