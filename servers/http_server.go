/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package servers

import (
	"strconv"
	"net/http"

	"github.com/mainflux/mainflux/controllers"
	"github.com/mainflux/mainflux/config"

	"github.com/go-zoo/bone"

)


func HttpServer(cfg config.Config) {

	mux := bone.New()

	/**
	 * Routes
	 */
	// Status
	mux.Get("/status", http.HandlerFunc(controllers.GetStatus))

	// Devices
	mux.Post("/devices", http.HandlerFunc(controllers.CreateDevice))
	mux.Get("/devices", http.HandlerFunc(controllers.GetDevices))

	mux.Get("/devices/:device_id", http.HandlerFunc(controllers.GetDevice))
	mux.Put("/devices/:device_id", http.HandlerFunc(controllers.UpdateDevice))

	mux.Delete("/devices/:device_id", http.HandlerFunc(controllers.DeleteDevice))

	// Channels
	mux.Post("/channels", http.HandlerFunc(controllers.CreateChannel))
	mux.Get("/channels", http.HandlerFunc(controllers.GetChannels))

	mux.Get("/channels/:channel_id", http.HandlerFunc(controllers.GetChannel))
	mux.Put("/channels/:channel_id", http.HandlerFunc(controllers.UpdateChannel))

	mux.Delete("/channels/:channel_id", http.HandlerFunc(controllers.DeleteChannel))

	/**
	 * Server
	 */
	http.ListenAndServe(cfg.HttpHost + ":" + strconv.Itoa(cfg.HttpPort), mux)

	// Use following to start HTTPS server on the same port
	//iris.ListenTLS(cfg.HttpHost + ":" + strconv.Itoa(cfg.HttpPort), "tls/mainflux.crt", "tls/mainflux.key")
}
